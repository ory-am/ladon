package sql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"github.com/rubenv/sql-migrate"

	"github.com/ory/common/compiler"
	"github.com/ory/common/pkg"
	"github.com/ory/ladon/access"
	"github.com/ory/ladon/manager"
	"github.com/ory/ladon/policy"
)

func init() {
	manager.DefaultManagers["sql"] = NewManager
}

var migrations = &migrate.MemoryMigrationSource{
	Migrations: []*migrate.Migration{
		&migrate.Migration{
			Id: "1",
			Up: []string{`CREATE TABLE IF NOT EXISTS ladon_policy (
	id           varchar(255) NOT NULL PRIMARY KEY,
	description  text NOT NULL,
	effect       text NOT NULL CHECK (effect='allow' OR effect='deny'),
	conditions 	 text NOT NULL
)`,
				`CREATE TABLE IF NOT EXISTS ladon_policy_subject (
	compiled text NOT NULL,
	template varchar(1023) NOT NULL,
	policy   varchar(255) NOT NULL,
	FOREIGN KEY (policy) REFERENCES ladon_policy(id) ON DELETE CASCADE
)`,
				`CREATE TABLE IF NOT EXISTS ladon_policy_permission (
	compiled text NOT NULL,
	template varchar(1023) NOT NULL,
	policy   varchar(255) NOT NULL,
	FOREIGN KEY (policy) REFERENCES ladon_policy(id) ON DELETE CASCADE
)`,
				`CREATE TABLE IF NOT EXISTS ladon_policy_resource (
	compiled text NOT NULL,
	template varchar(1023) NOT NULL,
	policy   varchar(255) NOT NULL,
	FOREIGN KEY (policy) REFERENCES ladon_policy(id) ON DELETE CASCADE
)`},
			Down: []string{
				"DROP TABLE ladon_policy",
				"DROP TABLE ladon_policy_subject",
				"DROP TABLE ladon_policy_permission",
				"DROP TABLE ladon_policy_resource",
			},
		},
	},
}

// SQLManager is an implementation for Manager to store policies persistently.
type SQLManager struct {
	db *sqlx.DB
}

// NewManager initializes a new SQLManager for given db instance.
func NewManager(opts ...manager.Option) (manager.Manager, error) {
	var o manager.Options
	for _, opt := range opts {
		opt(&o)
	}

	db, err := getSession(o)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if err := createSchemas(db); err != nil {
		return nil, errors.WithStack(err)
	}

	return &SQLManager{db: db}, nil
}

func getSession(o manager.Options) (*sqlx.DB, error) {
	if conn, ok := o.GetConnection(); ok {
		switch t := conn.(type) {
		case *sqlx.DB:
			return t, nil
		case *sql.DB:
			if o.Driver == "" {
				return nil, errors.New("Driver name required when using *sql.DB connection")
			}
			return sqlx.NewDb(t, o.Driver), nil
		default:
			err := fmt.Sprintf("Expected Connection argument of type %T or %T, got %T",
				&sql.DB{}, &sqlx.DB{}, t)
			return nil, errors.New(err)
		}
	}

	// Start new DB connection
	if o.Driver == "" {
		return nil, errors.New("Driver required for sql database connection")
	}
	// TODO: compose connection string from options if whole string not given
	if o.Connection == "" {
		return nil, errors.New("Connection string required for sql database connection")
	}
	ctx := context.Background()
	if o.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, o.Timeout)
		defer cancel()
	}
	return sqlx.ConnectContext(ctx, o.Driver, o.Connection)
}

// createSchemas creates ladon_policy tables
func createSchemas(db *sqlx.DB) error {
	n, err := migrate.Exec(db.DB, db.DriverName(), migrations, migrate.Up)
	if err != nil {
		return errors.Wrapf(err, "Could not migrate sql schema, applied %d migrations", n)
	}
	return nil
}

// Create inserts a new policy
func (s *SQLManager) Create(pol policy.Policy) (err error) {
	conditions := []byte("{}")
	if pol.GetConditions() != nil {
		cs := pol.GetConditions()
		conditions, err = json.Marshal(&cs)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	if tx, err := s.db.Begin(); err != nil {
		return errors.WithStack(err)
	} else if _, err = tx.Exec(s.db.Rebind("INSERT INTO ladon_policy (id, description, effect, conditions) VALUES (?, ?, ?, ?)"), pol.GetID(), pol.GetDescription(), pol.GetEffect(), conditions); err != nil {
		if err := tx.Rollback(); err != nil {
			return errors.WithStack(err)
		}
		return errors.WithStack(err)
	} else if err = createLinkSQL(s.db, tx, "ladon_policy_subject", pol, pol.GetSubjects()); err != nil {
		return err
	} else if err = createLinkSQL(s.db, tx, "ladon_policy_permission", pol, pol.GetActions()); err != nil {
		return err
	} else if err = createLinkSQL(s.db, tx, "ladon_policy_resource", pol, pol.GetResources()); err != nil {
		return err
	} else if err = tx.Commit(); err != nil {
		if err := tx.Rollback(); err != nil {
			return errors.WithStack(err)
		}
		return errors.WithStack(err)
	}

	return nil
}

// Get retrieves a policy.
func (s *SQLManager) Get(id string) (policy.Policy, error) {
	var p policy.DefaultPolicy
	var conditions []byte

	if err := s.db.QueryRow(s.db.Rebind("SELECT id, description, effect, conditions FROM ladon_policy WHERE id=?"), id).Scan(&p.ID, &p.Description, &p.Effect, &conditions); err == sql.ErrNoRows {
		return nil, pkg.ErrNotFound
	} else if err != nil {
		return nil, errors.WithStack(err)
	}

	p.Conditions = access.Conditions{}
	if err := json.Unmarshal(conditions, &p.Conditions); err != nil {
		return nil, errors.WithStack(err)
	}

	subjects, err := getLinkedSQL(s.db, "ladon_policy_subject", id)
	if err != nil {
		return nil, err
	}
	permissions, err := getLinkedSQL(s.db, "ladon_policy_permission", id)
	if err != nil {
		return nil, err
	}
	resources, err := getLinkedSQL(s.db, "ladon_policy_resource", id)
	if err != nil {
		return nil, err
	}

	p.Actions = permissions
	p.Subjects = subjects
	p.Resources = resources
	return &p, nil
}

// Delete removes a policy.
func (s *SQLManager) Delete(id string) error {
	_, err := s.db.Exec(s.db.Rebind("DELETE FROM ladon_policy WHERE id=?"), id)
	return errors.WithStack(err)
}

// FindPoliciesForSubject returns Policies (an array of policy) for a given subject
func (s *SQLManager) FindPoliciesForSubject(subject string) (policies policy.Policies, err error) {
	find := func(query string, args ...interface{}) (ids []string, err error) {
		rows, err := s.db.Query(query, args...)
		if err == sql.ErrNoRows {
			return nil, errors.Wrap(pkg.ErrNotFound, "")
		} else if err != nil {
			return nil, errors.WithStack(err)
		}
		defer rows.Close()
		for rows.Next() {
			var urn string
			if err = rows.Scan(&urn); err != nil {
				return nil, errors.WithStack(err)
			}
			ids = append(ids, urn)
		}
		return ids, nil
	}

	var query string
	switch s.db.DriverName() {
	case "postgres", "pgx":
		query = "SELECT policy FROM ladon_policy_subject WHERE $1 ~ ('^' || compiled || '$')"
	case "mysql":
		query = "SELECT policy FROM ladon_policy_subject WHERE ? REGEXP BINARY CONCAT('^', compiled, '$') GROUP BY policy"
	}

	if query == "" {
		return nil, errors.Errorf("driver %s not supported", s.db.DriverName())
	}

	subjects, err := find(query, subject)
	if err != nil {
		return policies, err
	}

	for _, id := range subjects {
		p, err := s.Get(id)
		if err != nil {
			return nil, err
		}
		policies = append(policies, p)
	}
	return policies, nil
}

func getLinkedSQL(db *sqlx.DB, table, policy string) ([]string, error) {
	urns := []string{}
	rows, err := db.Query(db.Rebind(fmt.Sprintf("SELECT template FROM %s WHERE policy=?", table)), policy)
	if err == sql.ErrNoRows {
		return nil, errors.Wrap(pkg.ErrNotFound, "")
	} else if err != nil {
		return nil, errors.WithStack(err)
	}

	defer rows.Close()
	for rows.Next() {
		var urn string
		if err = rows.Scan(&urn); err != nil {
			return []string{}, errors.WithStack(err)
		}
		urns = append(urns, urn)
	}
	return urns, nil
}

func createLinkSQL(db *sqlx.DB, tx *sql.Tx, table string, p policy.Policy, templates []string) error {
	for _, template := range templates {
		reg, err := compiler.CompileRegex(template, p.GetStartDelimiter(), p.GetEndDelimiter())

		// Execute SQL statement
		query := db.Rebind(fmt.Sprintf("INSERT INTO %s (policy, template, compiled) VALUES (?, ?, ?)", table))
		if _, err = tx.Exec(query, p.GetID(), template, reg.String()); err != nil {
			if rb := tx.Rollback(); rb != nil {
				return errors.WithStack(rb)
			}
			return errors.WithStack(err)
		}
	}
	return nil
}

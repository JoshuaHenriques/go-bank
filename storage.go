package main

import (
	"database/sql"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

type Storage interface {
	CreateAccount(*Account) error
	DeleteAccount(uuid.UUID) error
	UpdateAccount(*Account) error
	GetAccountByID(uuid.UUID) (*Account, error)
}

type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore() (*PostgresStore, error) {
	// connStr := "host=192.168.2.18:8080 user=postgres dbname=postgres password=postgres sslmode=verify-full"
	connStr := "postgresql://postgres:example@192.168.2.18?sslmode=disable"
	db, err := sql.Open("postgres", connStr)

	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &PostgresStore{
		db: db,
	}, nil
}

func (s *PostgresStore) Init() error {
	return s.createAccountTable()
}

func (s *PostgresStore) createAccountTable() error {
	query := `create table if not exists account (
		id uuid default uuid_generate_v4(),
		first_name varchar not null,
		last_name varchar not null,
		number serial,
		balance serial,
		created_at timestamp
	)`

	_, err := s.db.Exec(query)
	return err
}

func (s *PostgresStore) CreateAccount(*Account) error {
	return nil
}

func (s *PostgresStore) UpdateAccount(*Account) error {
	return nil
}

func (s *PostgresStore) DeleteAccount(id uuid.UUID) error {
	return nil
}

func (s *PostgresStore) GetAccountByID(id uuid.UUID) (*Account, error) {
	return nil, nil
}

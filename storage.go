package main

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

type Storage interface {
	CreateAccount(*Account) (uuid.UUID, error)
	DeleteAccount(uuid.UUID) error
	GetAccounts() ([]*Account, error)
	GetAccountByID(uuid.UUID) (*Account, error)
	GetAccountByEmail(string) (*Account, error)
}

type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore() (*PostgresStore, error) {
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
		email varchar not null,
		password varchar not null,
		number serial,
		balance serial,
		created_at timestamp
	)`

	_, err := s.db.Exec(query)
	return err
}

func (s *PostgresStore) CreateAccount(acc *Account) (uuid.UUID, error) {
	query := `insert into account (
		first_name, last_name, email, password, number, balance, created_at
	) values ($1, $2, $3, $4, $5, $6, $7) returning id`

	var id uuid.UUID
	err := s.db.QueryRow(query, acc.FirstName, acc.LastName, acc.Email, acc.Password, acc.Number, acc.Balance, acc.CreatedAt).Scan(&id)
	if err != nil {
		return uuid.Nil, err
	}

	return id, nil
}

func (s *PostgresStore) DeleteAccount(id uuid.UUID) error {
	// soft delete in production: mark account as deleted instead of hard deleting
	_, err := s.db.Query("delete from account where id = $1", id)
	return err
}

func (s *PostgresStore) GetAccountByEmail(email string) (*Account, error) {
	rows, err := s.db.Query("select * from account where email = $1", email)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		return scanIntoAccount(rows)
	}

	return nil, fmt.Errorf("account %s not found", email)
}

func (s *PostgresStore) GetAccountByID(id uuid.UUID) (*Account, error) {
	rows, err := s.db.Query("select * from account where id = $1", id)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		return scanIntoAccount(rows)
	}

	return nil, fmt.Errorf("account %s not found", id)
}

func (s *PostgresStore) GetAccounts() ([]*Account, error) {
	rows, err := s.db.Query("select * from account")
	if err != nil {
		return nil, err
	}

	accounts := []*Account{}
	for rows.Next() {
		account, err := scanIntoAccount(rows)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, account)
	}

	return accounts, nil
}

func scanIntoAccount(rows *sql.Rows) (*Account, error) {
	account := &Account{}

	err := rows.Scan(
		&account.ID,
		&account.FirstName,
		&account.LastName,
		&account.Email,
		&account.Password,
		&account.Number,
		&account.Balance,
		&account.CreatedAt)

	return account, err
}

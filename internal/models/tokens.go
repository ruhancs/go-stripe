package models

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"log"
	"time"
)

const (
	ScopeAuthentication = "authentication"
)

type Token struct {
	PlanText string `json:"token"`
	UserID int64 `json:"-"`
	Hash []byte `json:"-"`
	Expiry time.Time `json:"expiry"`
	Scope string `json:"-"`
}

func GenerateToken(userID int, ttl time.Duration, scope string) (*Token, error) {
	token := &Token{
		UserID: int64(userID),
		Expiry: time.Now().Add(ttl),
		Scope: scope,
	}

	randomBytes := make([]byte, 16)
	_,err := rand.Read(randomBytes)//criar dados aleatorios para gerar o token
	if err != nil {
		return nil, err
	}

	token.PlanText = base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(randomBytes)
	hash := sha256.Sum256(([]byte(token.PlanText)))
	token.Hash = hash[:]
	return token, nil
}

func (m *DbModel) InsertToken(t *Token, user User) error {
	ctx,cancel := context.WithTimeout(context.Background(), 3 * time.Second)
	defer cancel()

	//verificar se usuario ja tem token registrado e deletar o token existente
	stmt := `delete from tokens where user_id=?`
	_, err := m.DB.ExecContext(ctx,stmt, user.ID)
	if err != nil {
		return err
	}

	stmt = `insert into tokens (user_id, name, email, token_hash, expiry, created_at, updated_at)
		values(?,?,?,?,?,?,?)`
	
	_,err = m.DB.ExecContext(ctx, stmt,
		user.ID,
		user.LastName,
		user.Email,
		t.Hash,
		t.Expiry,
		time.Now(),
		time.Now(),
	)
	if err != nil {
		return err
	}
	return nil
}

func (m *DbModel) GetUserForToken(token string) (*User, error) {
	ctx,cancel := context.WithTimeout(context.Background(), 3 * time.Second)
	defer cancel()

	tokenHash := sha256.Sum256([]byte(token))
	var user User

	query := `
		select u.id, u.first_name, u.last_name, u.email from user u inner join tokens t on (u.id = t.user_id)
		where t.token_hash = ? and t.expiry > ?
	`
	err := m.DB.QueryRowContext(ctx,query,tokenHash[:], time.Now()).Scan(
		&user.ID,
		&user.FirstName,
		&user.LastName,
		&user.Email,
	)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return &user,nil
}
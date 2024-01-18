package main

import (
	"encoding/json"
	"errors"
	"os"
	"sync"
	"time"
)

type DB struct {
	path string
	mu *sync.RWMutex
}

type Chirp struct {
	AuthorId int `json:"author_id"`
	Body string `json:"body"`
	Id int `json:"id"`
}

type User struct {
	Id int `json:"id"`
	Password string `json:"password"`
	Email string `json:"email"`
	IsChirpyRed bool `json:"is_chirpy_red"`
}

type RefreshToken struct {
	Revoked bool
	Time time.Time
}

type DBStructure struct {
	Chirps map[int]Chirp
	Users map[int]User
	Tokens map[string]RefreshToken
}

var ErrNotExist = errors.New("resource does not exist")

func NewDB(path string) (*DB, error) {
	db := &DB{
		path: path,
		mu: &sync.RWMutex{},
	}

	return db, db.ensureDB()
}

func (db *DB) createDB() error {
	dbs := DBStructure{
		Chirps: make(map[int]Chirp),
		Users: make(map[int]User),
		Tokens: make(map[string]RefreshToken),
	}
	return db.writeDB(dbs)
}

func (db *DB) ensureDB() error {
	_, err := os.ReadFile(db.path)
	if os.IsNotExist(err) {
		return db.createDB()
	}
	return err
}

func (db *DB) CreateUser(email string, password string) (User, error) {
	dbs, err := db.loadDB()
	if err != nil { return User{}, err }

	id := len(dbs.Users) + 1
	
	if _, err := hasEmail(dbs, email); err == nil {
		return User{}, errors.New("email is already in user")
	}

	user := User{
		Email: email,
		Password: password,
		Id: id,
	}
	dbs.Users[id] = user

	err = db.writeDB(dbs)
	if err != nil { return User{}, err }

	return user, nil
}

func (db *DB) UpdateUser(id int, email, password string) (User, error) {
	dbs, err := db.loadDB()
	if err != nil { return User{}, err }

	user, ok := dbs.Users[id]
	if !ok { return User{}, ErrNotExist }

	user.Email = email
	user.Password = password
	dbs.Users[id] = user
	
	err = db.writeDB(dbs)
	if err != nil { return User{}, err }

	return user, nil
}

func (db *DB) UpgradeUser(id int) error {
	dbs, err := db.loadDB()
	if err != nil { return err }

	user, ok := dbs.Users[id]
	if !ok { return ErrNotExist }

	user.IsChirpyRed = true
	dbs.Users[id] = user

	return db.writeDB(dbs)
}

func (db *DB) RemoveUser(id int) error {
	dbs, err := db.loadDB()
	if err != nil { return err }

	delete(dbs.Users, id)

	return db.writeDB(dbs)
}

func (db *DB) GetUserFromId(id int) (User, error) {
	dbs, err := db.loadDB()
	if err != nil { return User{}, err}

	user, ok := dbs.Users[id]
	if !ok {
		return User{}, ErrNotExist
	}

	return user, nil
}

func (db *DB) GetUserFromEmail(email string) (User, error) {
	dbs, err := db.loadDB()
	if err != nil { return User{}, err }

	return hasEmail(dbs, email)
}

func (db *DB) CreateChirp(author int, body string) (Chirp, error) {

	dbs, err := db.loadDB()
	if err != nil { return Chirp{}, err }

	id := len(dbs.Chirps) + 1

	chirp := Chirp{
		AuthorId: author,
		Body: body,
		Id: id,
	}
	dbs.Chirps[id] = chirp

	err = db.writeDB(dbs)
	if err != nil { return Chirp{}, err }
	
	return chirp, nil
}

func (db *DB) GetChirp(id int) (Chirp, error) {
	dbs, err := db.loadDB()
	if err != nil { return Chirp{}, err }

	chirp, ok := dbs.Chirps[id]
	if !ok {
		return Chirp{}, ErrNotExist
	}

	return chirp, nil
}

func (db *DB) GetChirps() ([]Chirp, error) {
	dbs, err := db.loadDB()
	if err != nil { return nil, err }

	out := make([]Chirp, len(dbs.Chirps))

	for _, chirp := range dbs.Chirps {
		out[chirp.Id - 1] = chirp
	}
	return out, nil
}

func (db *DB) DeleteChirp(id int) error {
	dbs, err := db.loadDB()
	if err != nil { return err }

	delete(dbs.Chirps, id)

	return db.writeDB(dbs)
}

func (db *DB) IsChirpAuthor(author, id int) bool {
	chirp, err := db.GetChirp(id)
	if err != nil { return false }
	
	return chirp.AuthorId == author
}

func (db *DB) AddToken(token string) error {
	dbs, err := db.loadDB()
	if err != nil { return err }

	dbs.Tokens[token] = RefreshToken{ Revoked: false }
	return db.writeDB(dbs)
}

func (db *DB) RevokeToken(token string) error {
	dbs, err := db.loadDB()
	if err != nil { return err }

	dbs.Tokens[token] = RefreshToken{ Revoked: true, Time: time.Now()}
	return db.writeDB(dbs)
}

func (db *DB) ValidToken(token string) bool {
	dbs, err := db.loadDB()
	if err != nil { return false }

	refreshToken, ok := dbs.Tokens[token]
	if !ok {
		return true
	}
	if refreshToken.Revoked {
		return false
	}
	return true
}

func (db *DB) loadDB() (DBStructure, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	dat, err := os.ReadFile(db.path)
	if err != nil { return DBStructure{}, err }
	
	dbs := DBStructure{}

	err = json.Unmarshal(dat, &dbs)
	if err != nil { return DBStructure{}, err }
	return dbs, nil	
}

func (db *DB) writeDB(dbStructure DBStructure) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	dat, err := json.Marshal(dbStructure)
	if err != nil { return err }

	return os.WriteFile(db.path, dat, 0600)
}

func hasEmail(dbs DBStructure, email string) (User, error) {
	for _, user := range dbs.Users {
		if user.Email == email {
			return user, nil
		}
	}
	return User{}, ErrNotExist
}

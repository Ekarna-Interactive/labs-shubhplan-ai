package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/argon2"
)

type UserRole string

const (
	RoleAdmin   UserRole = "admin"
	RolePlanner UserRole = "planner"
	RoleGuest   UserRole = "guest"
)

type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"passwordHash"`
	FullName     string    `json:"fullName"`
	Role         UserRole  `json:"role"`
	CreatedAt    time.Time `json:"createdAt"`
}

type Session struct {
	Token     string    `json:"token"`
	UserID    string    `json:"userId"`
	UserEmail string    `json:"userEmail"`
	UserRole  UserRole  `json:"userRole"`
	ExpiresAt time.Time `json:"expiresAt"`
}

const (
	argonMemory      = 64 * 1024
	argonIterations  = 3
	argonParallelism = 2
	argonSaltLength  = 16
	argonKeyLength   = 32
)

func HashPasswordArgon2id(password string) (string, error) {
	salt := make([]byte, argonSaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	hash := argon2.IDKey([]byte(password), salt, argonIterations, argonMemory, argonParallelism, argonKeyLength)

	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		argonMemory, argonIterations, argonParallelism, b64Salt, b64Hash), nil
}

func decodeBase64Safe(s string) ([]byte, error) {
	sClean := strings.TrimRight(s, "=")
	if b, err := base64.RawStdEncoding.DecodeString(sClean); err == nil {
		return b, nil
	}
	return base64.StdEncoding.DecodeString(s)
}

func VerifyPasswordArgon2id(password, encodedHash string) (bool, error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false, errors.New("invalid argon2id hash format")
	}

	var memory uint32
	var iterations uint32
	var parallelism uint8
	_, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism)
	if err != nil {
		return false, err
	}

	salt, err := decodeBase64Safe(parts[4])
	if err != nil {
		return false, err
	}

	targetHash, err := decodeBase64Safe(parts[5])
	if err != nil {
		return false, err
	}

	computedHash := argon2.IDKey([]byte(password), salt, iterations, memory, parallelism, uint32(len(targetHash)))

	if subtle.ConstantTimeCompare(targetHash, computedHash) == 1 {
		return true, nil
	}
	return false, nil
}

type AuthManager struct {
	mu             sync.RWMutex
	users          map[string]User // key: email
	usersByID      map[string]User // key: id
	sessions       map[string]Session
	usersFilePath  string
}

var (
	defaultAuthManager *AuthManager
	authOnce           sync.Once
)

func GetAuthManager() *AuthManager {
	authOnce.Do(func() {
		dir := strings.TrimSpace(os.Getenv("SHUBH_DATA_DIR"))
		if dir == "" {
			dir = "./data"
		}
		defaultAuthManager = NewAuthManager(filepath.Join(dir, "users.json"))
	})
	return defaultAuthManager
}

func NewAuthManager(filePath string) *AuthManager {
	am := &AuthManager{
		users:         make(map[string]User),
		usersByID:     make(map[string]User),
		sessions:      make(map[string]Session),
		usersFilePath: filePath,
	}
	am.loadUsers()
	return am
}

func (am *AuthManager) loadUsers() {
	am.mu.Lock()
	defer am.mu.Unlock()

	_ = os.MkdirAll(filepath.Dir(am.usersFilePath), 0755)
	data, err := os.ReadFile(am.usersFilePath)
	if err != nil {
		return
	}

	var userList []User
	if err := json.Unmarshal(data, &userList); err == nil {
		for _, u := range userList {
			emailKey := strings.ToLower(u.Email)
			am.users[emailKey] = u
			am.usersByID[u.ID] = u
		}
		log.Printf("[Auth Manager] Loaded %d registered users from %s", len(userList), am.usersFilePath)
	}

	demoUsers := []struct {
		ID       string
		Email    string
		FullName string
		Role     UserRole
	}{
		{ID: "usr_demo_admin", Email: "admin@shubhplan.ai", FullName: "Demo Workspace Admin", Role: RoleAdmin},
		{ID: "usr_demo_planner", Email: "user@shubhplan.ai", FullName: "Demo Event Planner", Role: RolePlanner},
	}

	hash, _ := HashPasswordArgon2id("shubh2026")
	modified := false

	for _, d := range demoUsers {
		emailKey := strings.ToLower(d.Email)
		u, exists := am.users[emailKey]
		ok, _ := VerifyPasswordArgon2id("shubh2026", u.PasswordHash)
		if !exists || !ok {
			newUser := User{
				ID:           d.ID,
				Email:        d.Email,
				PasswordHash: hash,
				FullName:     d.FullName,
				Role:         d.Role,
				CreatedAt:    time.Now(),
			}
			am.users[emailKey] = newUser
			am.usersByID[newUser.ID] = newUser
			modified = true
			log.Printf("[Auth Manager] Pre-seeded verified demo account %s / shubh2026", d.Email)
		}
	}

	if modified {
		am.saveUsersLocked()
	}
}

func (am *AuthManager) saveUsersLocked() {
	userList := make([]User, 0, len(am.users))
	for _, u := range am.users {
		userList = append(userList, u)
	}
	data, err := json.MarshalIndent(userList, "", "  ")
	if err == nil {
		_ = os.WriteFile(am.usersFilePath, data, 0644)
	}
}

func (am *AuthManager) RegisterUser(email, password, fullName string, role UserRole) (User, error) {
	am.mu.Lock()
	defer am.mu.Unlock()

	emailClean := strings.ToLower(strings.TrimSpace(email))
	if emailClean == "" {
		return User{}, errors.New("email address is required")
	}
	if len(password) < 6 {
		return User{}, errors.New("password must be at least 6 characters")
	}
	if _, exists := am.users[emailClean]; exists {
		return User{}, errors.New("user with this email already exists")
	}

	hash, err := HashPasswordArgon2id(password)
	if err != nil {
		return User{}, fmt.Errorf("password hashing failed: %w", err)
	}

	if role == "" {
		role = RolePlanner
	}

	user := User{
		ID:           fmt.Sprintf("usr_%d", time.Now().UnixNano()),
		Email:        emailClean,
		PasswordHash: hash,
		FullName:     strings.TrimSpace(fullName),
		Role:         role,
		CreatedAt:    time.Now(),
	}

	am.users[emailClean] = user
	am.usersByID[user.ID] = user
	am.saveUsersLocked()

	log.Printf("[Auth Manager] Successfully registered user %s (%s)", user.Email, user.Role)
	return user, nil
}

func (am *AuthManager) AuthenticateUser(email, password string) (Session, error) {
	am.mu.Lock()
	defer am.mu.Unlock()

	emailClean := strings.ToLower(strings.TrimSpace(email))

	// Pre-seeded demo account fallback verification matching shubh-plan-open
	if (emailClean == "admin@shubhplan.ai" || emailClean == "user@shubhplan.ai") && password == "shubh2026" {
		role := RolePlanner
		id := "usr_demo_planner"
		name := "Demo Event Planner"
		if emailClean == "admin@shubhplan.ai" {
			role = RoleAdmin
			id = "usr_demo_admin"
			name = "Demo Workspace Admin"
		}
		hash, _ := HashPasswordArgon2id("shubh2026")
		user := User{
			ID:           id,
			Email:        emailClean,
			PasswordHash: hash,
			FullName:     name,
			Role:         role,
			CreatedAt:    time.Now(),
		}
		am.users[emailClean] = user
		am.usersByID[user.ID] = user
		am.saveUsersLocked()

		token := generateToken()
		session := Session{
			Token:     token,
			UserID:    user.ID,
			UserEmail: user.Email,
			UserRole:  user.Role,
			ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
		}
		am.sessions[token] = session
		log.Printf("[Auth Manager] Demo account %s authenticated cleanly", user.Email)
		return session, nil
	}

	user, exists := am.users[emailClean]
	if !exists {
		return Session{}, errors.New("invalid email or password")
	}

	ok, err := VerifyPasswordArgon2id(password, user.PasswordHash)
	if err != nil || !ok {
		return Session{}, errors.New("invalid email or password")
	}

	token := generateToken()
	session := Session{
		Token:     token,
		UserID:    user.ID,
		UserEmail: user.Email,
		UserRole:  user.Role,
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour), // 30 day session
	}

	am.sessions[token] = session
	log.Printf("[Auth Manager] User %s authenticated cleanly, session token generated", user.Email)
	return session, nil
}

func (am *AuthManager) CreateGuestDemoSession() Session {
	am.mu.Lock()
	defer am.mu.Unlock()

	guestID := fmt.Sprintf("guest_%d", time.Now().UnixNano())
	token := generateToken()

	session := Session{
		Token:     token,
		UserID:    guestID,
		UserEmail: fmt.Sprintf("%s@demo.shubhplan.local", guestID),
		UserRole:  RoleGuest,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}

	am.sessions[token] = session
	log.Printf("[Auth Manager] Created Instant Guest Demo session %s", guestID)
	return session
}

func (am *AuthManager) GetSession(token string) (Session, bool) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	if token == "" {
		return Session{}, false
	}

	sess, exists := am.sessions[token]
	if !exists {
		return Session{}, false
	}

	if time.Now().After(sess.ExpiresAt) {
		return Session{}, false
	}

	return sess, true
}

func (am *AuthManager) InvalidateSession(token string) {
	am.mu.Lock()
	defer am.mu.Unlock()
	delete(am.sessions, token)
}

func (am *AuthManager) DeleteUser(userID string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	user, exists := am.usersByID[userID]
	if exists {
		delete(am.users, strings.ToLower(user.Email))
		delete(am.usersByID, user.ID)
	}

	for token, sess := range am.sessions {
		if sess.UserID == userID {
			delete(am.sessions, token)
		}
	}

	if exists {
		am.saveUsersLocked()
		log.Printf("[Auth Manager] Permanently deleted user account %s (%s)", user.Email, user.ID)
	}
	return nil
}

func generateToken() string {
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

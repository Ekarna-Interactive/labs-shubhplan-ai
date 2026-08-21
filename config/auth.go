package config

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/argon2"
)

// UserRole defines access privileges
type UserRole string

const (
	RoleAdmin   UserRole = "admin"
	RolePlanner UserRole = "planner"
	RoleGuest   UserRole = "guest"
)

// User represents a registered application user
type User struct {
	ID            string    `json:"id"`
	Email         string    `json:"email"`
	PasswordHash  string    `json:"passwordHash"`
	FullName      string    `json:"fullName"`
	Role          UserRole  `json:"role"`
	SSHPublicKeys []string  `json:"sshPublicKeys,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
}

// Session represents an authenticated user session
type Session struct {
	Token     string    `json:"token"`
	UserID    string    `json:"userId"`
	UserEmail string    `json:"userEmail"`
	UserRole  UserRole  `json:"userRole"`
	ExpiresAt time.Time `json:"expiresAt"`
}

var (
	usersMu       sync.RWMutex
	activeSessions = make(map[string]Session)
	sessionMu      sync.RWMutex
)

// Argon2id parameters
const (
	argonMemory      = 64 * 1024
	argonIterations  = 3
	argonParallelism = 2
	argonSaltLength  = 16
	argonKeyLength   = 32
)

// HashPasswordArgon2id produces a secure Argon2id hash payload
func HashPasswordArgon2id(password string) (string, error) {
	salt := make([]byte, argonSaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	hash := argon2.IDKey([]byte(password), salt, argonIterations, argonMemory, argonParallelism, argonKeyLength)

	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	encoded := fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		argonMemory, argonIterations, argonParallelism, b64Salt, b64Hash)
	return encoded, nil
}

// VerifyPasswordArgon2id checks a plain password against an Argon2id hash payload
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

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, err
	}

	targetHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, err
	}

	hash := argon2.IDKey([]byte(password), salt, iterations, memory, parallelism, uint32(len(targetHash)))

	if subtle.ConstantTimeCompare(hash, targetHash) == 1 {
		return true, nil
	}
	return false, nil
}

func getUserStorePath() string {
	dataDir := getEnvClean("SHUBH_DATA_DIR")
	if dataDir == "" {
		dataDir = "./data"
	}
	_ = os.MkdirAll(dataDir, 0755)
	return filepath.Join(dataDir, "users.json")
}

// LoadUsers retrieves all registered users from storage
func LoadUsers() ([]User, error) {
	usersMu.RLock()
	defer usersMu.RUnlock()

	path := getUserStorePath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []User{}, nil
		}
		return nil, err
	}

	var users []User
	if err := json.Unmarshal(data, &users); err != nil {
		return []User{}, nil
	}
	return users, nil
}

// SaveUsers persists all user records to users.json with 0600 permissions
func SaveUsers(users []User) error {
	usersMu.Lock()
	defer usersMu.Unlock()

	path := getUserStorePath()
	data, err := json.MarshalIndent(users, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// IsSetupCompleted returns true if an Admin/Owner account already exists
func IsSetupCompleted() bool {
	users, err := LoadUsers()
	if err != nil || len(users) == 0 {
		return false
	}
	for _, u := range users {
		if u.Role == RoleAdmin {
			return true
		}
	}
	return false
}

// EnsureDefaultDemoUser seeds the pre-configured demo admin account if DEMO_MODE or SEED_DEMO_USER is active and no users exist
func EnsureDefaultDemoUser() {
	demoEnabled := getEnvClean("DEMO_MODE") == "true" || getEnvClean("DEMO_MODE") == "1" || getEnvClean("SEED_DEMO_USER") == "true" || getEnvClean("SEED_DEMO_USER") == "1"
	if demoEnabled && !IsSetupCompleted() {
		_, _ = CreateUser("admin@shubhplan.ai", "shubh2026", "Demo Workspace Owner", RoleAdmin)
	}
}

// CreateUser registers a new user record
func CreateUser(email, password, fullName string, role UserRole) (User, error) {
	cleanEmail := strings.ToLower(strings.TrimSpace(email))
	if cleanEmail == "" || password == "" {
		return User{}, errors.New("email and password are required")
	}

	users, _ := LoadUsers()
	for _, u := range users {
		if strings.EqualFold(u.Email, cleanEmail) {
			return User{}, errors.New("a user with this email already exists")
		}
	}

	hash, err := HashPasswordArgon2id(password)
	if err != nil {
		return User{}, fmt.Errorf("failed to hash password: %v", err)
	}

	b := make([]byte, 8)
	_, _ = rand.Read(b)
	userID := "usr_" + hex.EncodeToString(b)

	newUser := User{
		ID:           userID,
		Email:        cleanEmail,
		PasswordHash: hash,
		FullName:     strings.TrimSpace(fullName),
		Role:         role,
		CreatedAt:    time.Now(),
	}

	users = append(users, newUser)
	if err := SaveUsers(users); err != nil {
		return User{}, err
	}
	return newUser, nil
}

// AuthenticateUser verifies user credentials and creates a session
func AuthenticateUser(email, password string) (Session, error) {
	cleanEmail := strings.ToLower(strings.TrimSpace(email))
	users, err := LoadUsers()
	if err != nil || len(users) == 0 {
		return Session{}, errors.New("no registered users found")
	}

	var targetUser *User
	for i := range users {
		if strings.EqualFold(users[i].Email, cleanEmail) {
			targetUser = &users[i]
			break
		}
	}

	if targetUser == nil {
		return Session{}, errors.New("invalid email or password")
	}

	match, err := VerifyPasswordArgon2id(password, targetUser.PasswordHash)
	if err != nil || !match {
		return Session{}, errors.New("invalid email or password")
	}

	b := make([]byte, 32)
	_, _ = rand.Read(b)
	token := "shubh_auth_" + hex.EncodeToString(b)
	expires := time.Now().Add(7 * 24 * time.Hour)

	sess := Session{
		Token:     token,
		UserID:    targetUser.ID,
		UserEmail: targetUser.Email,
		UserRole:  targetUser.Role,
		ExpiresAt: expires,
	}

	sessionMu.Lock()
	activeSessions[token] = sess
	sessionMu.Unlock()

	return sess, nil
}

// ValidateSessionToken returns the active Session if valid
func ValidateSessionToken(token string) (Session, bool) {
	if token == "" {
		return Session{}, false
	}

	sessionMu.RLock()
	sess, exists := activeSessions[token]
	sessionMu.RUnlock()

	if !exists {
		return Session{}, false
	}

	if time.Now().After(sess.ExpiresAt) {
		sessionMu.Lock()
		delete(activeSessions, token)
		sessionMu.Unlock()
		return Session{}, false
	}
	return sess, true
}

// InvalidateSession removes an active session token
func InvalidateSession(token string) {
	sessionMu.Lock()
	delete(activeSessions, token)
	sessionMu.Unlock()
}

// GetUserBySSHKey maps an SSH Public Key to a registered user
func GetUserBySSHKey(pubKey string) (User, bool) {
	cleanKey := strings.TrimSpace(pubKey)
	if cleanKey == "" {
		return User{}, false
	}

	users, err := LoadUsers()
	if err != nil {
		return User{}, false
	}

	for _, u := range users {
		for _, key := range u.SSHPublicKeys {
			if strings.Contains(key, cleanKey) || strings.Contains(cleanKey, strings.TrimSpace(key)) {
				return u, true
			}
		}
	}
	return User{}, false
}

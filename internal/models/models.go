package models

import "time"

type Role string

const (
	RoleAdmin   Role = "admin"
	RoleEncoder Role = "encoder"
	RoleViewer  Role = "viewer"
)

type User struct {
	ID             int64
	Username       string
	PasswordHash   string
	DisplayName    string
	Role           Role
	ActiveBranchID int64
}

func (u User) CanWrite() bool {
	return u.Role == RoleAdmin || u.Role == RoleEncoder
}

func (u User) CanAdmin() bool {
	return u.Role == RoleAdmin
}

type Option struct {
	Value string
	Label string
}

type Record map[string]string

type DocumentListItem struct {
	ID        int64
	EntryID   string
	EntryDate time.Time
	Party     string
	Branch    string
	Reference string
	DRRef     string
	Status    string
	Net       string
	Encoder   string
}

package domain

import (
	"errors"
	"strings"
	"time"
)

type Contact struct {
	ID            int64     `json:"id"`
	UserID        int64     `json:"userId"`
	FirstName     string    `json:"firstName"`
	LastName      string    `json:"lastName"`
	Company       string    `json:"company,omitempty"`
	CompanyEmail  string    `json:"companyEmail,omitempty"`
	CompanyPhone  string    `json:"companyPhone,omitempty"`
	CompanyMobile string    `json:"companyMobile,omitempty"`
	Email         string    `json:"email,omitempty"`
	Phone         string    `json:"phone,omitempty"`
	Mobile        string    `json:"mobile,omitempty"`
	Address       string    `json:"address,omitempty"`
	Birthday      string    `json:"birthday,omitempty"`
	Notes         string    `json:"notes,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

func ValidateContact(contact Contact) error {
	if contact.UserID <= 0 {
		return errors.New("contact user is required")
	}
	if strings.TrimSpace(contact.FirstName) == "" && strings.TrimSpace(contact.LastName) == "" && strings.TrimSpace(contact.Company) == "" {
		return errors.New("contact name or company is required")
	}
	if len(contact.FirstName) > 120 || len(contact.LastName) > 120 || len(contact.Company) > 160 {
		return errors.New("contact name fields are too long")
	}
	if len(contact.Email) > 180 || len(contact.Phone) > 80 || len(contact.Mobile) > 80 {
		return errors.New("contact communication fields are too long")
	}
	if len(contact.CompanyEmail) > 180 || len(contact.CompanyPhone) > 80 || len(contact.CompanyMobile) > 80 {
		return errors.New("contact company communication fields are too long")
	}
	if len(contact.Address) > 1000 || len(contact.Notes) > 5000 {
		return errors.New("contact text fields are too long")
	}
	if contact.Birthday != "" {
		if _, err := time.Parse("2006-01-02", contact.Birthday); err != nil {
			return errors.New("contact birthday is invalid")
		}
	}
	return nil
}

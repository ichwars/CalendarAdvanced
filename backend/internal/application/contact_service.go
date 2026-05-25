package application

import (
	"fmt"

	"calendaradvanced/internal/domain"
	"calendaradvanced/internal/infrastructure/sqlite"
)

type ContactService struct {
	Store *sqlite.Store
	Audit *AuditService
}

type ContactInput struct {
	FirstName     string `json:"firstName"`
	LastName      string `json:"lastName"`
	Company       string `json:"company"`
	CompanyEmail  string `json:"companyEmail"`
	CompanyPhone  string `json:"companyPhone"`
	CompanyMobile string `json:"companyMobile"`
	Email         string `json:"email"`
	Phone         string `json:"phone"`
	Mobile        string `json:"mobile"`
	Address       string `json:"address"`
	Birthday      string `json:"birthday"`
	Notes         string `json:"notes"`
}

type ContactListInput struct {
	Query  string
	Limit  int
	Offset int
}

func (s *ContactService) List(user domain.User, input ContactListInput) ([]domain.Contact, error) {
	return s.Store.ListContacts(sqlite.ContactFilter{UserID: user.ID, Query: input.Query, Limit: input.Limit, Offset: input.Offset})
}

func (s *ContactService) Create(input ContactInput, user domain.User, ip, userAgent string) (domain.Contact, error) {
	contact := contactFromInput(input, user.ID)
	if err := domain.ValidateContact(contact); err != nil {
		return domain.Contact{}, NewError("invalid_contact", err.Error(), nil)
	}
	created, err := s.Store.CreateContact(contact)
	if err != nil {
		return domain.Contact{}, err
	}
	s.Audit.Record(user.ID, domain.AuditContactChanged, "contact", fmt.Sprint(created.ID), ip, userAgent, map[string]any{"operation": "create"})
	return created, nil
}

func (s *ContactService) Update(id int64, input ContactInput, user domain.User, ip, userAgent string) (domain.Contact, error) {
	contact := contactFromInput(input, user.ID)
	contact.ID = id
	if err := domain.ValidateContact(contact); err != nil {
		return domain.Contact{}, NewError("invalid_contact", err.Error(), nil)
	}
	updated, err := s.Store.UpdateContact(contact, user.ID)
	if err != nil {
		return domain.Contact{}, err
	}
	s.Audit.Record(user.ID, domain.AuditContactChanged, "contact", fmt.Sprint(updated.ID), ip, userAgent, map[string]any{"operation": "update"})
	return updated, nil
}

func (s *ContactService) Delete(id int64, user domain.User, ip, userAgent string) error {
	if err := s.Store.DeleteContact(id, user.ID); err != nil {
		return err
	}
	s.Audit.Record(user.ID, domain.AuditContactChanged, "contact", fmt.Sprint(id), ip, userAgent, map[string]any{"operation": "delete"})
	return nil
}

func contactFromInput(input ContactInput, userID int64) domain.Contact {
	return domain.Contact{
		UserID:        userID,
		FirstName:     input.FirstName,
		LastName:      input.LastName,
		Company:       input.Company,
		CompanyEmail:  input.CompanyEmail,
		CompanyPhone:  input.CompanyPhone,
		CompanyMobile: input.CompanyMobile,
		Email:         input.Email,
		Phone:         input.Phone,
		Mobile:        input.Mobile,
		Address:       input.Address,
		Birthday:      input.Birthday,
		Notes:         input.Notes,
	}
}

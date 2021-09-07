package main

import (
	"database/sql"
	"time"
)

type customer struct {
	ID                   int
	Name                 sql.NullString
	ArtificialPerson     sql.NullString
	Contact              sql.NullString
	Province             sql.NullString
	City                 sql.NullString
	District             sql.NullString
	Address              sql.NullString
	Email                sql.NullString
	URL                  sql.NullString
	Domain               sql.NullString
	BusinessScope        sql.NullString
	Type                 sql.NullString
	RegisteredCapital    sql.NullString
	FoundTime            time.Time
	SocialCrediCode      sql.NullString
	IndustryID           sql.NullInt64
	InsuredNumber        sql.NullInt64
	MxBrandID            sql.NullInt64
	MxBrandName          sql.NullString
	Mxrecord             sql.NullString
	IsSync               int
	MailTitle            sql.NullString
	SelfbuildBrandID     sql.NullInt64
	SelfbuildBrandName   sql.NullString
	Title                sql.NullString
	ContacttoolBrandID   sql.NullInt64
	ContacttoolBrandName sql.NullString
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

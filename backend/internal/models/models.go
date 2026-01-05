package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Agency struct {
	ID        uuid.UUID  `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	Name      string     `gorm:"size:500;not null" json:"name"`
	ShortName *string    `gorm:"size:255" json:"short_name,omitempty"`
	Slug      string     `gorm:"size:255;uniqueIndex;not null" json:"slug"`
	ParentID  *uuid.UUID `gorm:"type:uuid" json:"parent_id,omitempty"`
	Parent    *Agency    `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
	Children  []Agency   `gorm:"foreignKey:ParentID" json:"children,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type Title struct {
	ID               uuid.UUID  `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	Number           int        `gorm:"uniqueIndex;not null" json:"number"`
	Name             string     `gorm:"size:500;not null" json:"name"`
	LatestAmendedOn  *time.Time `json:"latest_amended_on,omitempty"`
	LatestIssueDate  *time.Time `json:"latest_issue_date,omitempty"`
	UpToDateAsOf     *time.Time `json:"up_to_date_as_of,omitempty"`
	Reserved         bool       `gorm:"default:false" json:"reserved"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type AgencyCFRReference struct {
	ID       uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	AgencyID uuid.UUID `gorm:"type:uuid;not null" json:"agency_id"`
	TitleID  uuid.UUID `gorm:"type:uuid;not null" json:"title_id"`
	Chapter  *string   `gorm:"size:50" json:"chapter,omitempty"`
	Agency   Agency    `gorm:"foreignKey:AgencyID" json:"agency"`
	Title    Title     `gorm:"foreignKey:TitleID" json:"title"`
}

type TitleContent struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	TitleID     uuid.UUID `gorm:"type:uuid;not null" json:"title_id"`
	ContentDate time.Time `gorm:"not null" json:"content_date"`
	XMLContent  string    `gorm:"type:text;not null" json:"xml_content"`
	WordCount   *int      `json:"word_count,omitempty"`
	Checksum    *string   `gorm:"size:64" json:"checksum,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	Title       Title     `gorm:"foreignKey:TitleID" json:"title"`
}

type HistoricalSnapshot struct {
	ID           uuid.UUID  `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	SnapshotDate time.Time  `gorm:"not null" json:"snapshot_date"`
	AgencyID     *uuid.UUID `gorm:"type:uuid" json:"agency_id,omitempty"`
	TitleID      *uuid.UUID `gorm:"type:uuid" json:"title_id,omitempty"`
	WordCount    *int       `json:"word_count,omitempty"`
	Checksum     *string    `gorm:"size:64" json:"checksum,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	Agency       *Agency    `gorm:"foreignKey:AgencyID" json:"agency,omitempty"`
	Title        *Title     `gorm:"foreignKey:TitleID" json:"title,omitempty"`
}

type AgencyChecksum struct {
	AgencyID    uuid.UUID `gorm:"type:uuid;primary_key" json:"agency_id"`
	Checksum    string    `gorm:"size:64;not null" json:"checksum"`
	ContentHash string    `gorm:"size:64;not null" json:"content_hash"`
	UpdatedAt   time.Time `gorm:"not null" json:"updated_at"`
	Agency      Agency    `gorm:"foreignKey:AgencyID" json:"agency"`
}

func (agency *Agency) BeforeCreate(tx *gorm.DB) error {
	if agency.ID == uuid.Nil {
		agency.ID = uuid.New()
	}
	return nil
}

func (title *Title) BeforeCreate(tx *gorm.DB) error {
	if title.ID == uuid.Nil {
		title.ID = uuid.New()
	}
	return nil
}

func (ref *AgencyCFRReference) BeforeCreate(tx *gorm.DB) error {
	if ref.ID == uuid.Nil {
		ref.ID = uuid.New()
	}
	return nil
}

func (content *TitleContent) BeforeCreate(tx *gorm.DB) error {
	if content.ID == uuid.Nil {
		content.ID = uuid.New()
	}
	return nil
}

func (snapshot *HistoricalSnapshot) BeforeCreate(tx *gorm.DB) error {
	if snapshot.ID == uuid.Nil {
		snapshot.ID = uuid.New()
	}
	return nil
}
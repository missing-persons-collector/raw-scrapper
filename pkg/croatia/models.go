package croatia

import (
	"gorm.io/datatypes"
	"missing-persons-scrapper/pkg/storage"
)

const Croatia_Scrapper_Table = "croatia_scrapped"
const Croatia_Images_Table = "croatia_images"

type DbImage struct {
	ID        int    `gorm:"column:id"`
	ItemID    int    `gorm:"column:item_id"`
	Extension string `gorm:"column:extension"`
	Blob      []byte `gorm:"column:blob"`
}

type RawData struct {
	ID               int
	Data             datatypes.JSON `gorm:"type:jsonb"`
	ItemID           string         `gorm:"column:item_id"`
	UniqueIdentifier string         `gorm:"column:unique_identifier;type:text"`
}

func NewRawData(data []byte, itemId, uniqueIdentifier string) RawData {
	return RawData{
		Data:             data,
		ItemID:           itemId,
		UniqueIdentifier: uniqueIdentifier,
	}
}

func NewDbImage(itemId int, extension string, blob []byte) DbImage {
	return DbImage{
		ItemID:    itemId,
		Blob:      blob,
		Extension: extension,
	}
}

func (RawData) TableName() string {
	return "croatia_scrapped"
}

func (DbImage) TableName() string {
	return "croatia_images"
}

func Migrate() error {
	if err := storage.DB.AutoMigrate(&RawData{}); err != nil {
		return err
	}

	if err := storage.DB.AutoMigrate(&DbImage{}); err != nil {
		return err
	}

	return nil
}

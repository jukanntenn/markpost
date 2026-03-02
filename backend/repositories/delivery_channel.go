package repositories

import (
	"errors"
	"markpost/models"

	"gorm.io/gorm"
)

type DeliveryChannelRepoInterface interface {
	ListByUserID(userID int) ([]models.DeliveryChannel, error)
	ListAll() ([]models.DeliveryChannel, error)
	GetByIDAndUserID(id int, userID int) (*models.DeliveryChannel, error)
	GetByID(id int) (*models.DeliveryChannel, error)
	Create(userID int, kind models.DeliveryChannelKind, name string, webhookURL string, keywords string, enabled bool) (*models.DeliveryChannel, error)
	Update(channel *models.DeliveryChannel) error
	DeleteByIDAndUserID(id int, userID int) (int64, error)
	DeleteByID(id int) (int64, error)
}

type DeliveryChannelRepo struct {
	database *models.Database
}

func NewDeliveryChannelRepo(database *models.Database) DeliveryChannelRepoInterface {
	return &DeliveryChannelRepo{database: database}
}

func (r *DeliveryChannelRepo) ListByUserID(userID int) ([]models.DeliveryChannel, error) {
	return models.GetDeliveryChannels(r.database, map[string]any{"user_id": userID})
}

func (r *DeliveryChannelRepo) ListAll() ([]models.DeliveryChannel, error) {
	db := r.database.DB()

	var channels []models.DeliveryChannel
	if err := db.Preload("User").Order("id asc").Find(&channels).Error; err != nil {
		return nil, err
	}
	return channels, nil
}

func (r *DeliveryChannelRepo) GetByIDAndUserID(id int, userID int) (*models.DeliveryChannel, error) {
	return models.GetDeliveryChannel(r.database, map[string]any{"id": id, "user_id": userID})
}

func (r *DeliveryChannelRepo) GetByID(id int) (*models.DeliveryChannel, error) {
	db := r.database.DB()

	var channel models.DeliveryChannel
	if err := db.Preload("User").Take(&channel, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, models.ErrNotFound
		}
		return nil, err
	}
	return &channel, nil
}

func (r *DeliveryChannelRepo) Create(userID int, kind models.DeliveryChannelKind, name string, webhookURL string, keywords string, enabled bool) (*models.DeliveryChannel, error) {
	channel := &models.DeliveryChannel{
		UserID:     userID,
		Kind:       kind,
		Name:       name,
		Enabled:    enabled,
		WebhookURL: webhookURL,
		Keywords:   keywords,
	}
	if err := channel.Create(r.database); err != nil {
		return nil, err
	}
	return channel, nil
}

func (r *DeliveryChannelRepo) Update(channel *models.DeliveryChannel) error {
	return channel.Update(r.database)
}

func (r *DeliveryChannelRepo) DeleteByIDAndUserID(id int, userID int) (int64, error) {
	return models.DeleteDeliveryChannels(r.database, map[string]any{"id": id, "user_id": userID})
}

func (r *DeliveryChannelRepo) DeleteByID(id int) (int64, error) {
	db := r.database.DB()
	tx := db.Delete(&models.DeliveryChannel{}, id)
	if tx.Error != nil {
		return 0, tx.Error
	}
	return tx.RowsAffected, nil
}

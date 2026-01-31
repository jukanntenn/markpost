package repositories

import (
	"markpost/models"
)

type DeliveryChannelRepoInterface interface {
	ListByUserID(userID int) ([]models.DeliveryChannel, error)
	GetByIDAndUserID(id int, userID int) (*models.DeliveryChannel, error)
	Create(userID int, kind models.DeliveryChannelKind, name string, webhookURL string, enabled bool) (*models.DeliveryChannel, error)
	Update(channel *models.DeliveryChannel) error
	DeleteByIDAndUserID(id int, userID int) (int64, error)
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

func (r *DeliveryChannelRepo) GetByIDAndUserID(id int, userID int) (*models.DeliveryChannel, error) {
	return models.GetDeliveryChannel(r.database, map[string]any{"id": id, "user_id": userID})
}

func (r *DeliveryChannelRepo) Create(userID int, kind models.DeliveryChannelKind, name string, webhookURL string, enabled bool) (*models.DeliveryChannel, error) {
	channel := &models.DeliveryChannel{
		UserID:     userID,
		Kind:       kind,
		Name:       name,
		Enabled:    enabled,
		WebhookURL: webhookURL,
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

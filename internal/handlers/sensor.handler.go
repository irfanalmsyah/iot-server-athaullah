package handlers

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/dafaath/iot-server/internal/dependencies"
	"github.com/dafaath/iot-server/internal/entities"
	"github.com/dafaath/iot-server/internal/helper"
	"github.com/dafaath/iot-server/internal/repositories"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SensorHandler struct {
	db                 *pgxpool.Pool
	repository         *repositories.SensorRepository
	hardwareRepository *repositories.HardwareRepository
	nodeRepository     *repositories.NodeRepository
	validator          *dependencies.Validator
}

func NewSensorHandler(db *pgxpool.Pool, sensorRepository *repositories.SensorRepository, hardwareRepository *repositories.HardwareRepository, nodeRepository *repositories.NodeRepository, validator *dependencies.Validator) (SensorHandler, error) {
	return SensorHandler{
		db:                 db,
		repository:         sensorRepository,
		hardwareRepository: hardwareRepository,
		nodeRepository:     nodeRepository,
		validator:          validator,
	}, nil
}

func (h *SensorHandler) CreateForm(c *fiber.Ctx) (err error) {
	return c.Render("sensor_form", fiber.Map{
		"title": "Create Sensor",
	}, "layouts/main")
}

func (h *SensorHandler) Create(c *fiber.Ctx) (err error) {
	ctx := context.Background()
	bodyPayload := entities.SensorCreate{}

	err = h.validator.ParseBody(c, &bodyPayload)
	if err != nil {
		return err
	}

	tx, err := h.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer helper.CommitOrRollback(ctx, tx, &err)

	node, err := h.nodeRepository.GetById(ctx, tx, bodyPayload.IdNode)
	if err != nil {
		return err
	}

	hardware, err := h.hardwareRepository.GetById(ctx, tx, bodyPayload.IdHardware)
	if err != nil {
		return err
	}

	hardwareType := strings.ToLower(hardware.Type)
	if hardwareType != "sensor" {
		return fiber.NewError(400, "Hardware type not match, type should be sensor")
	}

	currentUser, err := h.validator.GetAuthentication(c)
	if err != nil {
		return err
	}

	if currentUser.IdUser != node.IdUser {
		return fiber.NewError(403, "You can’t use other user’s node")
	}

	_, err = h.repository.Create(ctx, tx, &bodyPayload)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).SendString("Success add new sensor")
}

func (h *SensorHandler) GetAll(c *fiber.Ctx) (err error) {
	ctx := context.Background()
	tx, err := h.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer helper.CommitOrRollback(ctx, tx, &err)

	currentUser, err := h.validator.GetAuthentication(c)
	if err != nil {
		return err
	}

	sensors, err := h.repository.GetAll(ctx, tx, &currentUser)
	if err != nil {
		return err
	}

	accept := c.Accepts("application/json", "text/html")
	switch accept {
	case "text/html":
		return c.Render("sensor", fiber.Map{
			"title":   "Sensor",
			"sensors": sensors,
		}, "layouts/main")
	default:
		return c.Status(fiber.StatusOK).JSON(sensors)
	}
}

func (h *SensorHandler) GetById(c *fiber.Ctx) (err error) {
	ctx := context.Background()
	id, err := h.validator.ParseIdFromUrlParameter(c)
	if err != nil {
		return err
	}

	tx, err := h.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer helper.CommitOrRollback(ctx, tx, &err)

	sensor, err := h.repository.GetById(ctx, tx, id)
	if err != nil {
		return err
	}

	channels, err := h.repository.GetSensorChannel(ctx, tx, id)
	if err != nil {
		return err
	}

	sensorOwnerId, err := h.repository.GetIdUserWhoOwnSensorById(ctx, tx, id)
	if err != nil {
		return err
	}

	currentUser, err := h.validator.GetAuthentication(c)
	if err != nil {
		return err
	}

	if sensorOwnerId != currentUser.IdUser && !currentUser.IsAdmin {
		return fiber.NewError(403, "You can’t see another user’s sensor")
	}

	accept := c.Accepts("application/json", "text/html")
	switch accept {
	case "text/html":
		sort.Slice(channels, func(i, j int) bool {
			return channels[i].Time.Before(channels[j].Time)
		})

		return c.Render("sensor_detail", fiber.Map{
			"title":   "Sensor Detail",
			"sensor":  sensor,
			"channel": channels,
		}, "layouts/main")
	default:
		sensorWithChannelItem := entities.SensorWithChannel{
			Sensor:  sensor,
			Channel: channels,
		}
		return c.Status(fiber.StatusOK).JSON(sensorWithChannelItem)
	}
}

func (h *SensorHandler) UpdateForm(c *fiber.Ctx) (err error) {
	id, err := h.validator.ParseIdFromUrlParameter(c)
	if err != nil {
		return err
	}
	ctx := context.Background()

	tx, err := h.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer helper.CommitOrRollback(ctx, tx, &err)

	sensor, err := h.repository.GetById(ctx, tx, id)
	if err != nil {
		return err
	}

	return c.Render("sensor_form", fiber.Map{
		"title":  "Edit Sensor",
		"sensor": sensor,
		"edit":   true,
	}, "layouts/main")
}

func (h *SensorHandler) Update(c *fiber.Ctx) (err error) {
	ctx := context.Background()
	id, err := h.validator.ParseIdFromUrlParameter(c)
	if err != nil {
		return err
	}

	bodyPayload := &entities.SensorUpdate{}
	err = h.validator.ParseBody(c, bodyPayload)
	if err != nil {
		return err
	}

	tx, err := h.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer helper.CommitOrRollback(ctx, tx, &err)

	sensor, err := h.repository.GetById(ctx, tx, id)
	if err != nil {
		return err
	}

	sensorOwnerId, err := h.repository.GetIdUserWhoOwnSensorById(ctx, tx, id)
	if err != nil {
		return err
	}

	currentUser, err := h.validator.GetAuthentication(c)
	if err != nil {
		return err
	}

	if sensorOwnerId != currentUser.IdUser && !currentUser.IsAdmin {
		return fiber.NewError(403, "You can’t edit another user’s sensor")
	}

	err = h.repository.Update(ctx, tx, &sensor, bodyPayload)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).SendString("Success edit sensor")
}

func (h *SensorHandler) Delete(c *fiber.Ctx) (err error) {
	ctx := context.Background()
	id, err := h.validator.ParseIdFromUrlParameter(c)
	if err != nil {
		return err
	}

	tx, err := h.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer helper.CommitOrRollback(ctx, tx, &err)

	sensorOwnerId, err := h.repository.GetIdUserWhoOwnSensorById(ctx, tx, id)
	if err != nil {
		return err
	}

	currentUser, err := h.validator.GetAuthentication(c)
	if err != nil {
		return err
	}

	if sensorOwnerId != currentUser.IdUser && !currentUser.IsAdmin {
		return fiber.NewError(403, "You can't delete another user's sensor")
	}

	err = h.repository.Delete(ctx, tx, id)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).SendString(fmt.Sprintf("Success delete sensor, id: %d", id))
}

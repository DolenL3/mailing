package httpcontroller

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	swag "github.com/swaggest/swgui/v3"
	"go.uber.org/zap"

	"mailing/internal/config"
	"mailing/internal/mailing"
	"mailing/internal/models"
)

// HTTPController is http request controller.
type HTTPController struct {
	router  *gin.Engine
	service *mailing.MailingService
	config  *config.HTTPConfig
}

// New returns new HTTPController.
func New(router *gin.Engine, service *mailing.MailingService, config *config.HTTPConfig) *HTTPController {
	return &HTTPController{
		router:  router,
		service: service,
		config:  config,
	}
}

// Start starts http controller.
func (h *HTTPController) Start() error {
	l := zap.L()
	currentPath, err := os.Getwd()
	if err != nil {
		l.Warn("Can't get current path")
	}
	docsPath := currentPath + "/internal/docs/"
	h.router.StaticFile("/static/openapi.json", docsPath+"openapi.json")
	// Retrives all clients.
	h.router.GET("/clients", h.getClients)
	// Adds new client.
	h.router.POST("/clients", h.saveClient)
	// Changes existing client.
	h.router.PUT("/clients/:id", h.updateClient)
	// Deletes existing client.
	h.router.DELETE("clients/:id", h.deleteClient)

	// Retrives all mailings.
	h.router.GET("/mailings", h.getMailings)
	// Adds new mailing.
	h.router.POST("/mailings", h.saveMailing)
	// Changes existing mailing.
	h.router.PUT("/mailings/:id", h.updateMailing)
	// Deletes existing mailing.
	h.router.DELETE("/mailings/:id", h.deleteMailing)

	// Retrives common statistic for all mailings.
	h.router.GET("/mailings/statistic", h.commonStatistic)
	// Retrives detailed statistic for given mailing.
	h.router.GET("/mailings/statistic/:id", h.detailedStatistic)
	// Sends message to user.
	h.router.POST("/send/:id", h.sendMessage)
	// Opens API reference.
	h.router.GET("/docs/*any", gin.WrapH(swag.New("Mailing Service", "/static/openapi.json", "/docs/")))

	logger := zap.L()
	logger.Info(fmt.Sprintf("http server is up and running on %s", h.config.Host))
	err = h.router.Run(h.config.Host)
	if err != nil {
		return errors.Wrap(err, "run router")
	}
	return nil
}

// getClients retrives all clients.
// Doesn't convert it to DTO, altough should, so fields would be styled in json way :)
func (h *HTTPController) getClients(c *gin.Context) {
	clients, err := h.service.Storage.GetClients(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	clientsDTO := []*clientDTO{}
	for _, client := range clients {
		clientsDTO = append(clientsDTO, clientToDTO(client))
	}
	c.JSONP(http.StatusOK, clientsDTO)
}

// saveClient adds new client.
func (h *HTTPController) saveClient(c *gin.Context) {
	l := zap.L()
	client := clientDTO{}
	err := c.ShouldBind(&client)
	l.Info(fmt.Sprint(client))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err = h.service.Storage.SaveClient(c.Request.Context(), &models.Client{
		PhoneNumber:   client.PhoneNumber,
		PhoneOperator: client.PhoneOperator,
		Tag:           client.Tag,
		Timezone:      client.Timezone,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
}

// updateClient changes existing client.
func (h *HTTPController) updateClient(c *gin.Context) {
	idURL := c.Param("id")
	if idURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no id specified"})
		return
	}
	id, err := strconv.ParseInt(idURL, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	update := &clientUpdateDTO{}
	err = c.ShouldBind(update)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err = h.service.Storage.UpdateClient(c.Request.Context(), id, &models.ClientUpdate{
		PhoneNumber:   update.PhoneNumber,
		PhoneOperator: update.PhoneOperator,
		Tag:           update.Tag,
		Timezone:      update.Timezone,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
}

// deleteClient deletes existing client.
func (h *HTTPController) deleteClient(c *gin.Context) {
	idURL := c.Param("id")
	if idURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no id specified"})
		return
	}
	id, err := strconv.ParseInt(idURL, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err = h.service.Storage.DeleteClient(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
}

// getMailings retrives all mailings.
// Doesn't convert it to DTO, altough should, so fields would be styled in json way and the status would be human readable :)
func (h *HTTPController) getMailings(c *gin.Context) {
	mailings, err := h.service.Storage.GetMailings(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	mailingsDTO := []*mailingDTO{}
	for _, mailing := range mailings {
		mailingsDTO = append(mailingsDTO, mailingToDTO(mailing))
	}
	c.JSONP(http.StatusOK, mailingsDTO)
}

// saveMailing adds new mailing.
func (h *HTTPController) saveMailing(c *gin.Context) {
	msk, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	mailing := mailingDTO{}
	err = c.ShouldBind(&mailing)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var start, end time.Time
	if mailing.StartTime == "" {
		start = time.Now()
	} else {
		start, err = time.ParseInLocation("02-01-2006 15:04", mailing.StartTime, msk)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}
	if mailing.EndTime == "" {
		end = time.Now().Add(time.Hour)
	} else {
		end, err = time.ParseInLocation("02-01-2006 15:04", mailing.EndTime, msk)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}
	uuid := uuid.New()
	err = h.service.Storage.SaveMailing(c.Request.Context(), &models.Mailing{
		ID:   uuid,
		Text: mailing.Text,
		Filter: &models.Filter{
			PhoneOperator: mailing.Filter.PhoneOperator,
			Tag:           mailing.Filter.Tag,
			Timezone:      mailing.Filter.Timezone,
		},
		StartTime: start.UTC(),
		EndTime:   end.UTC(),
		Status:    models.MailingStatusPending,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
}

// updateMailing changes existing mailing.
func (h *HTTPController) updateMailing(c *gin.Context) {
	idURL := c.Param("id")
	if idURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no id specified"})
		return
	}
	id, err := uuid.Parse(idURL)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	update := mailingUpdateDTO{}
	err = c.ShouldBind(&update)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err = h.service.Storage.UpdateMailing(c.Request.Context(), id, &models.MailingUpdate{
		Text: update.Text,
		Filter: &models.Filter{
			PhoneOperator: update.Filter.PhoneOperator,
			Tag:           update.Filter.Tag,
			Timezone:      update.Filter.Timezone,
		},
		StartTime: update.StartTime,
		EndTime:   update.EndTime,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
}

// deleteMailing deletes existing mailing.
func (h *HTTPController) deleteMailing(c *gin.Context) {
	idURL := c.Param("id")
	if idURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no id specified"})
		return
	}
	id, err := uuid.Parse(idURL)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err = h.service.Storage.DeleteMailing(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
}

// commonStatistic retrives common statistic for all mailings.
func (h *HTTPController) commonStatistic(c *gin.Context) {
	statistic, err := h.service.Storage.CommonStatistic(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	statisticDTO := []*mailingStatisticDTO{}
	for _, stat := range statistic {
		statisticDTO = append(statisticDTO, mailingStatisticToDTO(stat))
	}
	c.JSONP(http.StatusOK, statisticDTO)
}

// detailedStatistic retrives detailed statistic for given mailing.
func (h *HTTPController) detailedStatistic(c *gin.Context) {
	idURL := c.Param("id")
	if idURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no id specified"})
		return
	}
	id, err := uuid.Parse(idURL)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	statistic, err := h.service.Storage.DetailedStatistic(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	statisticDTO := detailedStatisticToDTO(statistic)
	c.JSONP(http.StatusOK, statisticDTO)
}

// Configures active mailing.
// Sends message to user.
func (h *HTTPController) sendMessage(c *gin.Context) {
	l := zap.L()
	idURL := c.Param("id")
	if idURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no id specified"})
		return
	}
	id, err := strconv.ParseInt(idURL, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	msg := &models.Message{
		TimeStamp: time.Now(),
		Status:    models.SendStatusPending,
		MailingID: uuid.Nil,
		ClientID:  id,
	}
	msg.ID, err = h.service.Storage.SaveMessage(c.Request.Context(), msg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	type textStruct struct {
		Text string `json:"text"`
	}
	var text textStruct
	err = c.ShouldBind(&text)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	client, err := h.service.Storage.GetClientByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	err = h.service.MessageSender.Send(c.Request.Context(), msg.ID, client.PhoneNumber, text.Text)
	if err != nil {
		nestedErr := h.service.Storage.MarkMessage(c.Request.Context(), msg, models.SendStatusFailed)
		if nestedErr != nil {
			l.Error(fmt.Sprintf("FAIL: mark mailing status as failed\nError: %v", nestedErr))
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	err = h.service.Storage.MarkMessage(c.Request.Context(), msg, models.SendStatusSuccess)
	if err != nil {
		l.Error(fmt.Sprintf("FAIL: mark mailing status as success\nError: %v", err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
}

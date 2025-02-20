package services

import (
	"net/http"
	"sync"
	"time"

	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/rachel-lawrie/verus_backend_core/common"
	"github.com/rachel-lawrie/verus_backend_core/constants"
	"github.com/rachel-lawrie/verus_backend_core/models"
	"github.com/rachel-lawrie/verus_backend_core/utils"
	"github.com/rachel-lawrie/verus_backend_core/zaplogger"
	"go.mongodb.org/mongo-driver/bson"
	zap "go.uber.org/zap"
)

type ApplicantServiceImpl struct {
	CollectionName string
}

var (
	instance ApplicantServiceImpl
	once     sync.Once
)

func GetApplicantServiceImpl() ApplicantServiceImpl {
	once.Do(func() {
		instance = ApplicantServiceImpl{
			CollectionName: constants.CollectionApplicants,
		}
	})
	return instance
}

func (s *ApplicantServiceImpl) CreateApplicant(c *gin.Context, applicant *models.Applicant) (models.Applicant, error) {
	logger := zaplogger.GetLogger()
	collection := common.GetCollection(s.CollectionName)

	// Get the client ID from the context
	clientIDStr, err := utils.GetClientIDFromContext(c)
	if err != nil {
		return *applicant, err
	}

	applicant.ClientID = clientIDStr
	_, err = collection.InsertOne(c.Request.Context(), applicant)
	if err != nil {
		logger.Error("Error inserting applicant into MongoDB", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create applicant"})
		return *applicant, err
	}
	return *applicant, nil
}

func (s *ApplicantServiceImpl) GetAllApplicants(c *gin.Context) ([]models.Applicant, error) {
	logger := zaplogger.GetLogger()

	var applicants []models.Applicant

	collection := common.GetCollection(s.CollectionName)

	// Get the client ID from the context
	clientIDStr, err := utils.GetClientIDFromContext(c)
	if err != nil {
		return nil, err
	}

	cursor, err := collection.Find(c.Request.Context(), bson.M{"client_id": clientIDStr, "deleted": false})
	if err != nil {
		logger.Error("Error fetching applicants from MongoDB", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not fetch applicants"})
		return nil, err
	}
	defer cursor.Close(c.Request.Context())

	for cursor.Next(c.Request.Context()) {
		var applicant models.Applicant
		if err := cursor.Decode(&applicant); err != nil {
			logger.Error("Error decoding applicant from MongoDB", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error decoding applicant"})
			return nil, err
		}
		applicants = append(applicants, applicant)
	}

	if err := cursor.Err(); err != nil {
		logger.Error("Cursor error", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Cursor error"})
		return nil, err
	}

	// c.JSON(http.StatusOK, applicants)
	return applicants, nil
}

func (s *ApplicantServiceImpl) GetApplicant(c *gin.Context, applicantID string) (models.Applicant, error) {
	var applicant models.Applicant
	logger := zaplogger.GetLogger()

	// Get the client ID from the context
	clientIDStr, err := utils.GetClientIDFromContext(c)
	if err != nil {
		return applicant, err
	}

	// Generate the filter
	filter := bson.M{"client_id": clientIDStr, "applicant_id": applicantID, "deleted": false}

	// Fetch the applicant directly from the database
	collection := common.GetCollection(s.CollectionName)
	if collection == nil {
		err := fmt.Errorf("failed to get MongoDB collection: %s", s.CollectionName)
		logger.Error("Database collection not found", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not fetch applicant"})
		return applicant, err
	}

	// Perform the database query
	err = collection.FindOne(c.Request.Context(), filter).Decode(&applicant)
	if err != nil {
		logger.Error("Error fetching applicant from MongoDB", zap.Error(err), zap.String("applicantID", applicantID))
		c.JSON(http.StatusNotFound, gin.H{"error": "Applicant not found"})
		return applicant, err
	}

	// Add a debug log to inspect the fetched applicant record
	logger.Debug("Raw Applicant Record from Database", zap.Any("rawApplicant", applicant))
	return applicant, nil
}

func (s *ApplicantServiceImpl) UpdateApplicant(c *gin.Context, applicantID string, updates map[string]interface{}) (models.Applicant, error) {
	logger := zaplogger.GetLogger()
	var applicant models.Applicant

	// Get the client ID from the context
	clientIDStr, err := utils.GetClientIDFromContext(c)
	if err != nil {
		return applicant, err
	}

	filter, cacheKey, err := GenerateFilterAndCacheKey(applicantID, clientIDStr, s.CollectionName)

	if err != nil {
		logger.Error("Error generating filter and cache key", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not update applicant"})
		return applicant, err
	}

	// Build the update document
	updateDoc := bson.M{}
	for field, value := range updates {
		updateDoc[field] = value
	}
	updateDoc["updated_at"] = time.Now() // Always update the updated_at field

	update := bson.M{"$set": updateDoc}

	// Invalidate the cache after a successful update
	err = common.InvalidateCache(c, s.CollectionName, cacheKey, filter, update, nil)
	if err != nil {
		logger.Error("Error invalidating cache", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error invalidating cache"})
		return applicant, err
	}

	// Retrieve the updated document
	result, err := s.GetApplicant(c, applicantID)
	if err != nil {
		logger.Error("Error retrieving updated document", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not retrieve updated document"})
		return applicant, err
	}
	return result, err
}

// GenerateFilterAndCacheKey generates the filter and cache key for a document
func GenerateFilterAndCacheKey(applicantID, clientID, collectionName string) (bson.M, string, error) {
	logger := zaplogger.GetLogger()
	filter := bson.M{
		"applicant_id": applicantID,
		"client_id":    clientID,
		"deleted":      false,
	}
	cacheKey, err := common.GenerateCacheKey(collectionName, filter)
	if err != nil {
		logger.Error("Error generating cache key", zap.Error(err))
		return nil, "", err
	}
	return filter, cacheKey, nil
}

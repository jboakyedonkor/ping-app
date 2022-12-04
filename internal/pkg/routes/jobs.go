package routes

import (
	"net/http"

	"go.uber.org/zap"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jboakyedonkor/ping-app/internal/pkg/automators"
	"github.com/jboakyedonkor/ping-app/internal/pkg/cache"
)

type JobRoute struct {
	logger       *zap.SugaredLogger
	jobAutomator *automators.Automator
}

func (j *JobRoute) CreateJob(c *gin.Context) {
	var newJob automators.JobConfig
	if err := c.BindJSON(&newJob); err != nil {
		c.JSON(http.StatusBadRequest, GenericResponse{Message: "incorrect request body"})
		return
	}

	uid, err := j.jobAutomator.CreateNewJob(c.Request.Context(), newJob)
	if err != nil {
		c.JSON(http.StatusInternalServerError, GenericResponse{Message: "error creating new job"})
		return
	}

	c.JSON(http.StatusCreated, GenericResponse{UID: uid})
}

func (j *JobRoute) GetJobConfig(c *gin.Context) {
	id, ok := c.Params.Get("id")
	if !ok {
		c.JSON(http.StatusBadRequest, "no job uid specificed")
	}

	jobUUID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, GenericResponse{Message: "incorrect request body"})
		return
	}

	config, err := j.jobAutomator.GetJob(c.Request.Context(), jobUUID)
	if err != nil {
		if _, ok := err.(*cache.NotFoundError); ok {
			c.Status(http.StatusNotFound)
			return
		}

		c.JSON(http.StatusInternalServerError, GenericResponse{Message: "internal server error"})
		return
	}

	c.JSON(http.StatusOK, config)
}

func (j *JobRoute) DeleteJob(c *gin.Context) {
	id, ok := c.Params.Get("id")
	if !ok {
		c.JSON(http.StatusBadRequest, "no job uid specificed")
	}

	jobUUID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, GenericResponse{Message: "incorrect request body"})
		return
	}

	if err := j.jobAutomator.DeleteJob(c.Request.Context(), jobUUID); err != nil {
		c.JSON(http.StatusInternalServerError, GenericResponse{Message: "error deleting job"})
		return
	}
	c.Status(http.StatusNoContent)
}

func NewJobRoute(logger *zap.SugaredLogger, jobAutomator *automators.Automator) *JobRoute {
	return &JobRoute{
		logger:       logger,
		jobAutomator: jobAutomator,
	}
}

func (j *JobRoute) GetJobs(c *gin.Context) {
	ctx := c.Request.Context()
	configs, err := j.jobAutomator.GetRunningJobs(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, GenericResponse{Message: "internal server error"})
		return
	}

	c.JSON(http.StatusOK, configs)

}

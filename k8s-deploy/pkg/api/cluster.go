/*
 * Copyright 2019-2020 VMware, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 * http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package api

import (
	"errors"

	"github.com/FederatedAI/KubeFATE/k8s-deploy/pkg/job"
	"github.com/FederatedAI/KubeFATE/k8s-deploy/pkg/modules"
	"github.com/FederatedAI/KubeFATE/k8s-deploy/pkg/service"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// Cluster api of cluster
type Cluster struct {
}

// Router is cluster router definition method
func (c *Cluster) Router(r *gin.RouterGroup) {

	authMiddleware, _ := GetAuthMiddleware()
	cluster := r.Group("/cluster")
	cluster.Use(authMiddleware.MiddlewareFunc())
	{
		cluster.POST("", c.createCluster)
		cluster.PUT("", c.setCluster)
		cluster.GET("/", c.getClusterList)
		cluster.GET("/:clusterId", c.getCluster)
		cluster.DELETE("/:clusterId", c.deleteCluster)
	}
}

// createCluster Create a new cluster
// @Summary Create a new cluster
// @Tags Cluster
// @Produce  json
// @Param  ClusterArgs body job.ClusterArgs true "Cluster Args"
// @Success 200 {object} JSONResult{data=modules.Job} "Success"
// @Failure 400 {object} JSONERRORResult "Bad Request"
// @Failure 401 {object} JSONERRORResult "Unauthorized operation"
// @Failure 500 {object} JSONERRORResult "Internal server error"
// @Router /cluster [post]
// @Param Authorization header string true "Authentication header"
// @Security ApiKeyAuth
// @Security OAuth2Application[write, admin]
func (*Cluster) createCluster(c *gin.Context) {

	user, _ := c.Get(identityKey)

	clusterArgs := new(job.ClusterArgs)

	if err := c.ShouldBindJSON(&clusterArgs); err != nil {
		log.Error().Err(err).Msg("request error")
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	log.Debug().Interface("parameters", clusterArgs).Msg("parameters")

	// create job and use goroutine do job result save to db
	j, err := job.ClusterInstall(clusterArgs, user.(*User).Username)
	if err != nil {
		log.Error().Err(err).Msg("request error")
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	log.Debug().Interface("data", j).Msg("createCluster success")
	c.JSON(200, gin.H{"msg": "createCluster success", "data": j})
}

// setCluster Updates a cluster in the store with form data
// @Summary Updates a cluster in the store with form data
// @Tags Cluster
// @Produce  json
// @Param  ClusterArgs body job.ClusterArgs true "Cluster Args"
// @Success 200 {object} JSONResult{data=modules.Job} "Success"
// @Failure 400 {object} JSONERRORResult "Bad Request"
// @Failure 401 {object} JSONERRORResult "Unauthorized operation"
// @Failure 500 {object} JSONERRORResult "Internal server error"
// @Router /cluster [put]
// @Param Authorization header string true "Authentication header"
// @Security ApiKeyAuth
// @Security OAuth2Application[write, admin]
func (*Cluster) setCluster(c *gin.Context) {

	//cluster := new(db.Cluster)
	//if err := c.ShouldBindJSON(&cluster); err != nil {
	//	c.JSON(400, gin.H{"error": err.Error()})
	//	return
	//}

	user, _ := c.Get(identityKey)

	clusterArgs := new(job.ClusterArgs)

	if err := c.ShouldBindJSON(&clusterArgs); err != nil {
		log.Error().Err(err).Msg("request error")
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	log.Debug().Interface("parameters", clusterArgs).Msg("parameters")

	// create job and use goroutine do job result save to db
	j, err := job.ClusterUpdate(clusterArgs, user.(*User).Username)
	if err != nil {
		log.Error().Err(err).Msg("request error")
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	log.Debug().Interface("data", j).Msg("setCluster success")
	c.JSON(200, gin.H{"msg": "setCluster success", "data": j})
}

// getCluster create a Cluster
// @Summary create Cluster
// @Tags Cluster
// @Produce  json
// @Param   clusterId   path  string  true  "cluster Id"
// @Success 200 {object} JSONResult{data=modules.Cluster} "Success"
// @Failure 400 {object} JSONERRORResult "Bad Request"
// @Failure 401 {object} JSONERRORResult "Unauthorized operation"
// @Failure 500 {object} JSONERRORResult "Internal server error"
// @Router /cluster/{clusterId} [get]
// @Param Authorization header string true "Authentication header"
// @Security ApiKeyAuth
// @Security OAuth2Application[write, admin]
func (*Cluster) getCluster(c *gin.Context) {

	clusterID := c.Param("clusterId")
	if clusterID == "" {
		log.Error().Err(errors.New("not exit clusterId")).Msg("request error")
		c.JSON(400, gin.H{"error": "not exit clusterId"})
		return
	}

	hc := modules.Cluster{Uuid: clusterID}
	cluster, err := hc.Get()
	if err != nil {
		log.Error().Err(err).Str("uuid", clusterID).Msg("get cluster error")
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	cluster.Info, err = service.GetClusterInfo(cluster.Name, cluster.NameSpace)
	if err != nil {
		log.Error().Err(err).Str("Name", cluster.Name).Str("NameSpace", cluster.NameSpace).Msg("GetClusterInfo error")
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	if cluster.Spec == nil {
		cluster.Spec = make(map[string]interface{})
	}

	clusterStatus, err := cluster.GetClusterStatus()
	if err != nil {
		log.Error().Err(err).Str("Name", cluster.Name).Str("NameSpace", cluster.NameSpace).Msg("GetClusterStatus error")
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	cluster.Info["status"] = clusterStatus

	podsStatus, exists := clusterStatus["modules"]
	if exists && podsStatus != nil {
		for _, v := range podsStatus {
			if v == "Pending" {
				cluster.Status = modules.ClusterStatusPending
				continue
			} else if v == "Unknown" {
				cluster.Status = modules.ClusterStatusUnknown
				continue
			} else if v == "Failed" {
				cluster.Status = modules.ClusterStatusFailed
				break
			}
		}
	}
	log.Debug().Interface("data", cluster).Msg("getCluster success")
	c.JSON(200, gin.H{"msg": "getCluster success", "data": &cluster})
}

// getClusterList List all available clusters
// @Summary List all available clusters
// @Tags Cluster
// @Produce  json
// @Param   all     query    boolean      true        "get All cluster"
// @Success 200 {object} JSONResult{data=[]modules.Cluster} "Success"
// @Failure 400 {object} JSONERRORResult "Bad Request"
// @Failure 401 {object} JSONERRORResult "Unauthorized operation"
// @Failure 500 {object} JSONERRORResult "Internal server error"
// @Router /cluster/ [get]
// @Param Authorization header string true "Authentication header"
// @Security ApiKeyAuth
// @Security OAuth2Application[write, admin]
func (*Cluster) getClusterList(c *gin.Context) {

	all := false
	qall := c.Query("all")
	if qall == "true" {
		all = true
	}

	log.Debug().Bool("all", all).Msg("get args")

	clusterList, err := new(modules.Cluster).GetListAll(all)

	if err != nil {
		log.Error().Err(err).Msg("request error")
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	log.Debug().Interface("data", clusterList).Msg("getClusterList success")
	c.JSON(200, gin.H{"msg": "getClusterList success", "data": clusterList})
}

// deleteCluster Delete a cluster
// @Summary Delete a cluster
// @Tags Cluster
// @Produce  json
// @Param   clusterId   path  string  true  "cluster Id"
// @Success 200 {object} JSONResult{data=modules.Job} "Success"
// @Failure 400 {object} JSONERRORResult "Bad Request"
// @Failure 401 {object} JSONERRORResult "Unauthorized operation"
// @Failure 500 {object} JSONERRORResult "Internal server error"
// @Router /cluster/{clusterId} [delete]
// @Param Authorization header string true "Authentication header"
// @Security ApiKeyAuth
// @Security OAuth2Application[write, admin]
func (*Cluster) deleteCluster(c *gin.Context) {

	user, _ := c.Get(identityKey)

	clusterID := c.Param("clusterId")
	if clusterID == "" {
		log.Error().Err(errors.New("not exit clusterId")).Msg("request error")
		c.JSON(400, gin.H{"error": "not exit clusterId"})
	}

	j, err := job.ClusterDelete(clusterID, user.(*User).Username)
	if err != nil {
		log.Error().Err(err).Msg("request error")
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	log.Debug().Interface("data", j).Msg("deleteCluster success")
	c.JSON(200, gin.H{"msg": "deleteCluster success", "data": j})
}

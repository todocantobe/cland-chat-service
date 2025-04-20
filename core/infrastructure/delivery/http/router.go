package router

import (
	"github.com/gin-gonic/gin"
	"sync"
)

var (
	once   sync.Once
	router *gin.Engine
)

func GetRouter() *gin.Engine {
	once.Do(func() {
		router = gin.Default()
	})
	return router
}

func init() {

}

package route

import (
	"cafe/controller"
	"cafe/utils"
	"github.com/gin-gonic/gin"
)

func CafeRoutes(router *gin.Engine) {
	cafeGroup := router.Group("/cafe")
	cafeGroup.Use(utils.CafeMiddleware())
	{
		cafeGroup.PUT("/update", controller.UpdateMyCafe)
		cafeGroup.GET("/my-cafe", controller.GetMyCafe)
		cafeGroup.GET("/foods/get-my", controller.GetMyCafeFoods)
		cafeGroup.POST("/foods/add", controller.AddFood)
		cafeGroup.POST("/foods/add/excel", controller.BulkAddFood)
		cafeGroup.PUT("/foods/update/:id", controller.UpdateFood)
		cafeGroup.DELETE("/foods/delete/:id", controller.DeleteFood)
		cafeGroup.POST("/cafe/category/add", controller.AddCategory)
		cafeGroup.PUT("/cafe/category/update/:id", controller.UpdateCategory)
		cafeGroup.DELETE("/cafe/category/delete/:id", controller.DeleteCategory)
		cafeGroup.GET("/cafe/categories/get-my", controller.GetMyCategories)
	}
	router.POST("/cafe/refresh-token", controller.RefreshTokenFunc)
	router.POST("/cafe/auth/login", controller.LoginManager)
	router.GET("/cafe/categories//categories/:cafe_id", controller.GetCategoriesByCafeID)
	router.GET("/cafe/categories/foods", controller.GetCafeCategoriesWithFoods)
	router.GET("/cafe/foods/by-category", controller.GetFoodsByCategoryID)
	router.GET("/cafe/foods/:id", controller.GetFoodByID)
}

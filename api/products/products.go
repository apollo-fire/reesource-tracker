package products

import (
	"database/sql"
	"net/http"
	"reesource-tracker/api/sync"
	"reesource-tracker/lib/database"
	id_helper "reesource-tracker/lib/id_helper"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func Routes(route *gin.RouterGroup) {
	route.GET("/products", getProducts)
	route.POST("/product", createProduct)
	route.GET("/product/:product_id", getProduct)
	route.POST("/product/:product_id", updateProduct)
	route.DELETE("/product/:product_id", deleteProduct)
}

// DELETE /product/:product_id
func deleteProduct(c *gin.Context) {
	productID := c.Param("product_id")
	if productID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "product_id required"})
		return
	}
	binary_uuid, errMsg, ok := id_helper.MustParseAndMarshalUUID(productID)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": errMsg})
		return
	}
	err := database.Connection.DeleteProductByID(c, binary_uuid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
	sync.BroadcastEvent("products_updated", gin.H{})
}

func createProduct(c *gin.Context) {
	var req struct {
		Name            string  `json:"name"`
		ParentProductID *string `json:"parent_product_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	new_uid, err := uuid.New().MarshalBinary()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate product ID"})
		return
	}
	
	// Convert parent product ID if provided
	var parentProductIDBytes []byte
	if req.ParentProductID != nil {
		var ok bool
		parentProductIDBytes, _, ok = id_helper.MustParseAndMarshalUUID(*req.ParentProductID)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid parent product ID format"})
			return
		}
	}
	
	// UpsertProduct expects UpsertProductParams struct
	params := database.UpsertProductParams{
		ID:              new_uid,
		Name:            req.Name,
		ParentProductID: parentProductIDBytes,
	}
	err = database.Connection.UpsertProduct(c, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success"})
	sync.BroadcastEvent("products_updated", gin.H{})
}

func getProduct(c *gin.Context) {
	productID := c.Param("product_id")
	if productID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "product_id required"})
		return
	}
	productIDBytes, errMsg, ok := id_helper.MustParseAndMarshalUUID(productID)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": errMsg})
		return
	}
	product, err := database.Connection.GetProductByID(c, productIDBytes)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, product)
}

func updateProduct(c *gin.Context) {
	productID := c.Param("product_id")
	if productID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "product_id required"})
		return
	}
	var req struct {
		Name            string `json:"name"`
		ParentProductID string `json:"parent_product_id"`
		PartNumber      string `json:"part_number"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	binary_uuid, errMsg, ok := id_helper.MustParseAndMarshalUUID(productID)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": errMsg})
		return
	}
	var parentBinaryUUID []byte
	if req.ParentProductID != "" {
		parentBinaryUUID, errMsg, ok = id_helper.MustParseAndMarshalUUID(req.ParentProductID)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": errMsg})
			return
		}
	}

	// UpsertProduct expects UpsertProductParams struct
	params := database.UpsertProductParams{
		ID:              binary_uuid,
		Name:            req.Name,
		ParentProductID: parentBinaryUUID,
		PartNumber:      sql.NullString{String: req.PartNumber, Valid: true},
	}
	err := database.Connection.UpsertProduct(c, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success"})
	sync.BroadcastEvent("products_updated", gin.H{})
}

func getProducts(c *gin.Context) {
	res, err := database.Connection.GetProducts(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

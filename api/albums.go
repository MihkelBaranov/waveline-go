package api
import (
	"net/http"
	"github.com/labstack/echo"
)

func Albums(c echo.Context) error {
	response := make(map[string]interface{})
	response["message"] = "ALBUMS"
	return c.JSON(http.StatusOK, response)
}

func Album(c echo.Context) error {
	response := make(map[string]interface{})
	response["message"] = "ALBUM"
	return c.JSON(http.StatusOK, response)
}

func New(c echo.Context) error {
	response := make(map[string]interface{})
	response["message"] = "NEW ALBUMS"
	return c.JSON(http.StatusOK, response)
}
package provider

import (
	"fmt"
	"net/http"

	"github.com/gruntwork-io/terragrunt/terraform/registry/router"
	"github.com/julienschmidt/httprouter"
)

const (
	defaultURLPrefix = "providers"
)

type Controller struct {
	urlPrefix string
}

func NewController() *Controller {
	return &Controller{
		urlPrefix: defaultURLPrefix,
	}
}

func (contr *Controller) Prefix() string {
	return contr.urlPrefix
}

func (contr *Controller) Subscribe(endpointer router.Endpointer) {
	endpointer.Get("/", contr.indexAction)
}

func (contr *Controller) indexAction(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	fmt.Fprint(w, "Welcome!\n")
}

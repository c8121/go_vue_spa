package go_vue_spa

import (
	"io/fs"
	"net/http"
	"path"
	"strings"

	auth "github.com/c8121/go_vue_spa/auth/gin"
	"github.com/c8121/go_vue_spa/util"
	"github.com/gin-gonic/gin"
)

type Resource struct {
	RootFs          fs.FS
	SubDirName      string
	DefaultDocument string

	httpFS http.FileSystem
}

func (r *Resource) GetHttpFS() http.FileSystem {
	if r.httpFS == nil {
		r.httpFS = http.FS(r.RootFs)
	}
	return r.httpFS
}

func CreateRoutesFromFS(router *gin.Engine, resources []Resource) {

	router.NoRoute(auth.UserAuthRequiredHandler(func(c *gin.Context) {

		requestPath := c.Request.URL.Path

		for _, resource := range resources {

			httpFS := resource.GetHttpFS()
			var fileSystemPath string

			if requestPath == "/" {

				if resource.DefaultDocument != "" {
					fileSystemPath = path.Join(resource.SubDirName, resource.DefaultDocument)
					file, err := httpFS.Open(fileSystemPath)
					if err == nil {
						defer util.CloseOrLog(file)
						stat, err := file.Stat()
						if err == nil {
							http.ServeContent(c.Writer, c.Request, stat.Name(), stat.ModTime(), file)
							return
						}
					}
				}

			} else {

				requestPath = path.Clean(requestPath)
				requestPath = strings.TrimPrefix(requestPath, "/")

				fileSystemPath = path.Join(resource.SubDirName, requestPath)
				if fileSystemPath != "" {
					stat, err := fs.Stat(resource.RootFs, fileSystemPath)
					if err == nil {
						if stat.Mode().IsRegular() {
							c.FileFromFS(fileSystemPath, httpFS)
							return
						}
					}
				}
			}
		}

		http.NotFound(c.Writer, c.Request)
	}))

}

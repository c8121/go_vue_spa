package go_vue_spa

import (
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/c8121/go_vue_spa/auth/gin"
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

		path := c.Request.URL.Path

		for _, resource := range resources {
			httpFS := resource.GetHttpFS()
			var fspath string
			if path == "/" {

				if resource.DefaultDocument != "" {
					fspath = filepath.Join(resource.SubDirName, resource.DefaultDocument)
					file, err := httpFS.Open(fspath)
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

				fspath = filepath.Join(resource.SubDirName, strings.TrimPrefix(path, "/"))
				if fspath != "" {
					stat, err := fs.Stat(resource.RootFs, fspath)
					if err == nil {
						if stat.Mode().IsRegular() {
							c.FileFromFS(fspath, httpFS)
							return
						}
					}
				}
			}
		}

		http.NotFound(c.Writer, c.Request)
	}))

}

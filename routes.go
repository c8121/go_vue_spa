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
	SubDirName      string //Subdirectory of RootFs to Serve from
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

		for _, resource := range resources {
			if ServeFromResource(c, resource) {
				return
			}
		}

		http.NotFound(c.Writer, c.Request)
	}))

}

func ServeFromResource(c *gin.Context, resource Resource) bool {

	httpFS := resource.GetHttpFS()
	var fileSystemPath string

	requestPath := cleanPath(c.Request.URL.Path)
	fileSystemPath = path.Join(cleanPath(resource.SubDirName), requestPath)
	if fileSystemPath == "" {
		fileSystemPath = "."
	}

	stat, err := fs.Stat(resource.RootFs, fileSystemPath)
	if err == nil {
		if stat.Mode().IsRegular() {
			file, err := httpFS.Open(fileSystemPath)
			if err == nil {
				defer util.CloseOrLog(file)
				http.ServeContent(c.Writer, c.Request, stat.Name(), stat.ModTime(), file)
				return true
			}
		} else if resource.DefaultDocument != "" && stat.Mode().IsDir() {
			fileSystemPath = path.Join(fileSystemPath, resource.DefaultDocument)
			stat, err = fs.Stat(resource.RootFs, fileSystemPath)
			if err == nil {
				file, err := httpFS.Open(fileSystemPath)
				if err == nil {
					defer util.CloseOrLog(file)
					http.ServeContent(c.Writer, c.Request, stat.Name(), stat.ModTime(), file)
					return true
				}
			}
		}
	}
	return false
}

func cleanPath(p string) string {

	p = path.Clean(p)
	p = strings.TrimPrefix(p, "/")

	if p != "." {
		return p
	}
	return ""
}

// BoltInspect project boltinspect.go
package boltinspect

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/boltdb/bolt"
)

type BoltInspect struct {
	db        *bolt.DB
	Templates *template.Template
}

func New(db *bolt.DB) *BoltInspect {
	tpls := template.New("inspect.tpl")
	template.Must(tpls.Parse(inspectTemplate))
	template.Must(tpls.New("view.tpl").Parse(viewTemplate))

	return &BoltInspect{
		db:        db,
		Templates: tpls,
	}
}

type bucketData struct {
	ID string
}

type itemData struct {
	ID      string
	Content string
}

func (bi *BoltInspect) InspectEndpoint(rw http.ResponseWriter, req *http.Request) {
	static := req.FormValue("static")
	if static != "" {
		switch static {
		case "skeleton":
			skeleton.ServeHTTP(rw, req)
		case "normalize":
			normalize.ServeHTTP(rw, req)
		}
		return
	}

	bucket := req.FormValue("bucket")
	buckets := strings.Split(bucket, "/")
	bucketsbyte := make([][]byte, 0)
	for _, v := range buckets {
		if v != "" {
			bucketsbyte = append(bucketsbyte, []byte(v))
		}
	}

	if req.Method == "POST" {
		action := req.FormValue("action")
		if action == "delete" {
			bi.db.Update(func(tx *bolt.Tx) error {
				if len(bucketsbyte) == 1 {
					return tx.DeleteBucket(bucketsbyte[0])
				}
				bucket := tx.Bucket(bucketsbyte[0])
				for _, b := range bucketsbyte[1 : len(bucketsbyte)-1] {
					bucket = bucket.Bucket(b)
				}
				last := bucketsbyte[len(bucketsbyte)-1]
				if bucket.Bucket(last) != nil {
					return bucket.DeleteBucket(last)
				}
				return bucket.Delete(last)
			})
		}
		path := strings.Join(buckets[:len(buckets)-1], "/")
		http.Redirect(rw, req, "?bucket="+path, http.StatusSeeOther)
		return
	}

	bucketdata := make([]*bucketData, 0)
	itemdata := make([]*itemData, 0)
	fmt.Println(bucketsbyte)
	bi.db.View(func(tx *bolt.Tx) error {
		if len(bucketsbyte) == 0 {
			tx.ForEach(func(k []byte, v *bolt.Bucket) error {
				bucketdata = append(bucketdata, &bucketData{
					ID: string(k),
				})
				return nil
			})
			bi.Templates.ExecuteTemplate(rw, "inspect.tpl", struct {
				Items   []*bucketData
			}{
				bucketdata,
			})
			return nil
		}
		bucket := tx.Bucket(bucketsbyte[0])
		idPrefix := string(bucketsbyte[0])
		for _, b := range bucketsbyte[1:] {
			if bucket.Get(b) == nil {
				bucket = bucket.Bucket(b)
				idPrefix = idPrefix + "/" + string(b)
			} else {
				bi.Templates.ExecuteTemplate(rw, "view.tpl", &itemData{
					ID:      idPrefix + "/" + string(b),
					Content: string(bucket.Get(b)),
				})
				return nil
			}
		}
		bucket.ForEach(func(k, v []byte) error {
			bucketdata = append(bucketdata, &bucketData{
				ID: idPrefix + "/" + string(k),
			})
			return nil
		})
		bi.Templates.ExecuteTemplate(rw, "inspect.tpl", struct {
			Items []*bucketData
		}{
			bucketdata,
		})
		return nil

	})

}

var inspectTemplate = `
<html><head>
  <link rel="stylesheet" href="?static=normalize">
  <link rel="stylesheet" href="?static=skeleton">
</head><body>
<div class="container">
<table class="u-full-width">
<thead>
	<tr>
		<th>Name</th>
		<th>Actions</th>
	</tr>
</thead>
<tbody>
	{{range .Items}}
	<tr>
		<td><a href="?bucket={{.ID}}">{{.ID}}</a></td>
		<td>
			<form action="" method="POST">
				<button class="button-primary" name="action" value="delete">Delete</button>
			</form>
		</td>
	</tr>
	{{end}}
</tbody>
</table>
</div>
</body></html>
`

var viewTemplate = `
<html><head></head><body>
<h1>{{.ID}}</h1>
{{.Content}}
</body></html>
`

func css(a asset) http.Handler {
	return a
}

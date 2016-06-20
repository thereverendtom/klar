package clair

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/optiopay/klar/docker"
)

const (
	imageName     = "test-image"
	imageTag      = "image-tag"
	imageRegistry = "https://image-registry"
	layerHash     = "blob1"
)

func TestAnalyse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		responseFile := "testdata/clair-get"

		if r.Method == "POST" {
			var envelope layerEnvelope
			if err := json.NewDecoder(r.Body).Decode(&envelope); err != nil {
				http.Error(w, `{"message": "json decode"}`, http.StatusBadRequest)
				return
			}
			layer := envelope.Layer
			if layer.Name != layerHash {
				http.Error(w, `{"message": "layer name"}`, http.StatusBadRequest)
				return
			}
			if layer.Path != fmt.Sprintf("%s/%s/blobs/%s", imageRegistry, imageName, layerHash) {
				http.Error(w, `{"message": "layer path"}`, http.StatusBadRequest)
				return
			}

			if layer.ParentName != "" && layer.ParentName != layerHash {
				http.Error(w, `{"message": "layer parent name"}`, http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusCreated)
			responseFile = "testdata/clair-post"
		} else {
			if r.URL.Path != fmt.Sprintf("/v1/layers/%s", layerHash) {
				http.Error(w, `{"message": "get path"}`, http.StatusBadRequest)
				return
			}
		}

		resp, err := ioutil.ReadFile(responseFile)
		if err != nil {
			t.Fatalf("Can't load clair test response %s", err.Error())
		}
		fmt.Fprintln(w, string(resp))
	}))

	defer ts.Close()

	dockerImage := &docker.Image{
		Registry: imageRegistry,
		Name:     imageName,
		Tag:      imageTag,
		FsLayers: []docker.FsLayer{
			docker.FsLayer{layerHash},
			docker.FsLayer{layerHash},
		},
	}

	c := NewClair(ts.URL)
	vs, err := c.Analyse(dockerImage)
	if err != nil {
		t.Fatalf("Analyse failed: %s", err.Error())
	}
	if len(vs) != 2 {
		t.Fatalf("Expected 2 vulnerabilities, got %d", len(vs))
	}
}

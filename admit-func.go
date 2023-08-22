package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/sirupsen/logrus"
	admv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

func writeErr(err error, w io.Writer) {
	lg.WithError(err).Error("writing error response")
	json.NewEncoder(w).Encode(admv1.AdmissionResponse{
		Result: &metav1.Status{
			Message: err.Error(),
		},
	})
}

var (
	scheme = runtime.NewScheme()
	codecs = serializer.NewCodecFactory(scheme)
)

type admitFunc func(admv1.AdmissionReview) (*admv1.AdmissionResponse, error)

func (a admitFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Body == nil {
		writeErr(fmt.Errorf("no body"), w)
		return
	}
	defer r.Body.Close()

	bs, err := io.ReadAll(r.Body)
	if err != nil {
		writeErr(err, w)
		return
	}

	var ar admv1.AdmissionReview
	_, _, err = codecs.UniversalDeserializer().Decode(bs, nil, &ar)
	if err != nil {
		writeErr(err, w)
		return
	}

	res, err := a(ar)
	if err != nil {
		writeErr(err, w)
		return
	}

	if ar.Request != nil {
		res.UID = ar.Request.UID
	}
	ar = admv1.AdmissionReview{
		TypeMeta: ar.TypeMeta,
		Response: res,
	}

	lg.WithField("res", ar).Info("writing response")

	err = json.NewEncoder(w).Encode(ar)
	if err != nil {
		logrus.WithError(err).Error("could not serialize admissionreview")
	}
}

package main

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	admv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	label      = "microcumul.us/injectssl"
	volumeName = "microcumulus-injected-ssl"
)

// Type for less ugly jsonpatch
type p struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

// We make lots of these.
type m map[string]any

var (
	ctrDeletes = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ca_injector_pods_deleted",
		Help: "The number of pods deleted by the ca-injector pod",
	}, []string{"namespace", "name"})

	ctrPatches = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ca_injector_pods_mutated",
		Help: "The number of pods mutated by the ca-injector webhook",
	}, []string{"namespace"})
)

func main() {
	cfg := setupConfig()
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	f, err := os.OpenFile(cfg.GetString("tls.crt"), os.O_RDONLY, 0400)
	if err != nil {
		lg.WithError(err).Fatal("could not read tls cert for serving and to check expiry")
	}

	cert, err := getFirstExpiringCert(f)
	if err != nil {
		lg.WithError(err).Fatal("could not read cert end date for certificate")
	}

	go func() {
		time.Sleep(time.Until(cert.NotAfter))
		ioutil.WriteFile("/dev/termination-log", []byte("shutting down due to expired certificate, hoping it has been refreshed"), 0600)
		lg.Fatal("cert expired; shutting down")
	}()

	sslFileName := path.Join("/ssl", cfg.GetString("tls.ca.key"))
	lg.WithField("file", sslFileName).Info("generated ssl filename")

	http.Handle("/metrics", promhttp.Handler())

	http.Handle("/pods", admitFunc(func(ar admv1.AdmissionReview) (res *admv1.AdmissionResponse, err error) {
		var pod corev1.Pod
		obj, _, err := codecs.UniversalDeserializer().Decode(ar.Request.Object.Raw, nil, &pod)
		if err != nil {
			if err != nil {
				lg.WithError(err).Error("could not deserialize pod spec")
				return nil, err
			}
		}

		lg := lg.WithFields(logrus.Fields{
			"ar.Request.Name":                        ar.Request.Name,
			"ar.Request.Namespace":                   ar.Request.Namespace,
			"pod.Name":                               pod.Name,
			"pod.Namespace":                          pod.Namespace,
			"pod.CreationTimestamp":                  pod.CreationTimestamp.Time,
			"obj.GetObjectKind().GroupVersionKind()": obj.GetObjectKind().GroupVersionKind(),
		})

		if pod.Annotations[label] == "" {
			lg.Info("allowing")
			return &admv1.AdmissionResponse{
				Allowed: true,
			}, nil
		}
		lg.Info("will patch")

		var patch []p
		if pod.Spec.Volumes == nil {
			patch = append(patch, p{
				Op:    "add",
				Path:  "/spec/volumes",
				Value: []any{}, // add array if none
			})
		}

		patch = append(patch, p{
			Op:   "add",
			Path: "/spec/volumes/-",
			Value: m{
				"name": volumeName,
				"secret": m{
					"secretName": pod.Annotations[label],
				},
			},
		})

		for i, ctr := range pod.Spec.Containers {
			ps := []p{{
				Op:   "add",
				Path: fmt.Sprintf("/spec/containers/%d/env/-", i),
				Value: m{
					"name":  "SSL_CERT_FILE",
					"value": sslFileName,
				},
			}, {
				Op:   "add",
				Path: fmt.Sprintf("/spec/containers/%d/env/-", i),
				Value: m{
					"name":  "NODE_EXTRA_CA_CERTS",
					"value": sslFileName,
				},
			}, {
				Op:   "add",
				Path: fmt.Sprintf("/spec/containers/%d/volumeMounts/-", i),
				Value: m{
					"name":      volumeName,
					"mountPath": "/ssl",
					"readOnly":  true,
				},
			}}

			if ctr.Env == nil {
				ps = append([]p{{
					Op:    "add",
					Path:  fmt.Sprintf("/spec/containers/%d/env", i),
					Value: []any{}, //add the array if none
				}}, ps...)
			}
			if len(ctr.VolumeMounts) == 0 {
				ps = append([]p{{
					Op:    "add",
					Path:  fmt.Sprintf("/spec/containers/%d/volumeMounts", i),
					Value: []any{}, //add the array if none
				}}, ps...)
			}

			patch = append(patch, ps...)
		}

		ctrPatches.WithLabelValues(ar.Request.Name).Inc()
		lg.WithField("patch", patch).Info("patching")

		bs, _ := json.Marshal(patch)

		pt := admv1.PatchTypeJSONPatch
		return &admv1.AdmissionResponse{
			Allowed:   true,
			Patch:     bs,
			PatchType: &pt,
			Result: &metav1.Status{
				Message: "modified",
			},
		}, nil
	}))

	conf, err := rest.InClusterConfig()
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		time.Sleep(5 * time.Second)

		f := false
		for {
			if f {
				time.Sleep(60 * time.Second)
			}
			f = true

			cs := kubernetes.NewForConfigOrDie(conf)
			pods, err := cs.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
			if err != nil {
				logrus.WithError(err).Fatal("error listing pods")
			}

			lg.WithField("len(pods.Items)", len(pods.Items)).Info("got pod list")

		items:
			for _, pod := range pods.Items {
				lg := lg.WithFields(logrus.Fields{
					"pod.Name":      pod.Name,
					"pod.Namespace": pod.Namespace,
				})

				or := corev1.ObjectReference{
					Kind:            pod.Kind,
					Namespace:       pod.Namespace,
					Name:            pod.Name,
					UID:             pod.UID,
					APIVersion:      pod.APIVersion,
					ResourceVersion: pod.ResourceVersion,
				}

				if len(pod.OwnerReferences) > 0 {
					or = corev1.ObjectReference{
						Kind:       pod.OwnerReferences[0].Kind,
						Namespace:  pod.Namespace,
						Name:       pod.OwnerReferences[0].Name,
						UID:        pod.OwnerReferences[0].UID,
						APIVersion: pod.OwnerReferences[0].APIVersion,
					}
				}

				secret := pod.Annotations[label]
				if secret == "" {
					lg.Debug("did not find annotation " + label)
					continue
				}

				// Look for well-known volume in list of mounts
				for _, vol := range pod.Spec.Volumes {
					if vol.Secret != nil && vol.Secret.SecretName == secret && vol.Name == volumeName {
						lg.Debug("found volume matching secret from annotation")
						continue items
					}
				}

				lg.Info("deleting pod; CA mount not found")

				_, err = cs.CoreV1().Events(pod.Namespace).Create(ctx, &corev1.Event{
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "ca-injector-delete-",
					},
					LastTimestamp:       metav1.Now(),
					ReportingController: "ca-injector",
					InvolvedObject:      or,
					Reason:              "CertAuthorityMissing",
					Message:             fmt.Sprintf("pod annotation on %q has not been applied by ca-injector mutatingadmissionwebhook; pod will be deleted", pod.Name),
					Type:                "Warning",
				}, metav1.CreateOptions{})
				if err != nil {
					lg.WithError(err).Error("error generating pod deletion event")
				}

				ctrDeletes.WithLabelValues(pod.Namespace, pod.GenerateName).Inc()

				err := cs.CoreV1().Pods(pod.Namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{})
				if err != nil {
					logrus.WithError(err).WithField("pod", pod.Name).Error("error deleting pod")
				}
			}
		}
	}()

	s := http.Server{
		Addr:              ":8443",
		Handler:           http.DefaultServeMux,
		ReadHeaderTimeout: 30 * time.Second,
	}

	go func() {
		// Shutdown listener
		<-ctx.Done()
		ctx, cancel := context.WithTimeout(context.Background(), cfg.GetDuration("shutdown.timeout"))
		defer cancel()
		s.Shutdown(ctx)
	}()

	lg.Info("listening")

	lg.Fatal(s.ListenAndServeTLS(cfg.GetString("tls.crt"), cfg.GetString("tls.key")))
}

func getFirstExpiringCert(r io.Reader) (*x509.Certificate, error) {
	bs, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("error getting bytes for cert: %w", err)
	}

	var first *x509.Certificate
	for {
		var blk *pem.Block
		blk, bs = pem.Decode(bs)
		if blk == nil {
			return first, nil
		}
		cert, err := x509.ParseCertificate(blk.Bytes)
		if err != nil {
			return nil, fmt.Errorf("error parsing cert in chain: %w", err)
		}
		if first == nil || cert.NotAfter.Before(first.NotAfter) {
			first = cert
		}
	}
}

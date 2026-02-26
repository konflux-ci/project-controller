package eventr_test

import (
	"errors"
	"fmt"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/events"

	"github.com/konflux-ci/project-controller/pkg/logr/eventr"
)

var _ = Describe("Eventr", func() {
	var (
		recorder            *events.FakeRecorder
		object              runtime.Object
		logger              logr.Logger
		someErr             error
		expectRecorderEvent func(eventType string, reason string, message string)
	)

	BeforeEach(func() {
		recorder = events.NewFakeRecorder(10)

		object = &corev1.ConfigMap{
			TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "ConfigMap"},
			ObjectMeta: metav1.ObjectMeta{Name: "cm1"},
		}
		Expect(object.GetObjectKind().GroupVersionKind().GroupKind().String()).NotTo(BeEmpty())

		expectRecorderEvent = func(eventType, reason, message string) {
			GinkgoHelper()
			Expect(recorder.Events).Should(Receive(Equal(
				fmt.Sprintf("%s %s %s", eventType, reason, message),
			)))
		}

		someErr = errors.New("some error")

		logger = eventr.NewEventr(recorder, object)
	})

	It("Generates events on subject on logging calls", func() {
		logger.Info("Something happened")

		expectRecorderEvent("Normal", "Info", "Something happened")
	})
	It("Emits 'Warning' events on logged errors", func() {
		logger.Error(someErr, "error happened")

		expectRecorderEvent("Warning", "Info", "error happened: some error")
	})
	It("Allows customizing 'reason' via 'eventReason' keyword arg", func() {
		logger.Info("Something happened", "eventReason", "Something")

		expectRecorderEvent("Normal", "Something", "Something happened")

		logger.Error(someErr, "error happened", "eventReason", "Something")

		expectRecorderEvent("Warning", "Something", "error happened: some error")
	})
	It("Supports setting 'eventReason' via WithValues", func() {
		logWReason := logger.WithValues("eventReason", "EmbeddedReason")
		logWReason.Info("Something happened")

		expectRecorderEvent("Normal", "EmbeddedReason", "Something happened")

		logWReason.Error(someErr, "error happened")

		expectRecorderEvent("Warning", "EmbeddedReason", "error happened: some error")
	})
	It("Only reports level 0 messages", func() {
		logger.V(1).Info("not important")

		Expect(recorder.Events).ShouldNot(Receive())
	})
})

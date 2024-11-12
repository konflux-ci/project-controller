/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"go/build"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	projctlv1beta1 "github.com/konflux-ci/project-controller/api/v1beta1"
	"github.com/konflux-ci/project-controller/pkg/testhelpers"

	// Depend on the Application/Component API so we can get the CRD files
	applicaitonapiv1alpha1 "github.com/konflux-ci/application-api/api/v1alpha1"
	imagectrlapiv1alpha1 "github.com/konflux-ci/image-controller/api/v1alpha1"
	intgtstscnariov1beta2 "github.com/konflux-ci/integration-service/api/v1beta2"
	releasev1alpha1 "github.com/konflux-ci/release-service/api/v1alpha1"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client
var saClient client.Client
var saCluster cluster.Cluster
var testEnv *envtest.Environment

func TestControllers(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Controller Suite")
}

func apiObjCrdPath(apiObj interface{}) string {
	var err error
	var appApiSrcImport *build.Package

	appApiPkgPath := reflect.TypeOf(apiObj).PkgPath()
	appApiSrcImport, err = build.Default.Import(appApiPkgPath, "", build.FindOnly)
	Expect(err).NotTo(HaveOccurred())

	return filepath.Join(appApiSrcImport.Dir, "..", "..", "config", "crd", "bases")
}

var _ = BeforeSuite(func() {
	var err error

	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	By("bootstrapping test environment")

	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "..", "config", "crd", "bases"),
			apiObjCrdPath(applicaitonapiv1alpha1.Application{}),
			apiObjCrdPath(imagectrlapiv1alpha1.ImageRepository{}),
			apiObjCrdPath(intgtstscnariov1beta2.IntegrationTestScenario{}),
			apiObjCrdPath(releasev1alpha1.ReleasePlan{}),
		},
		ErrorIfCRDPathMissing: true,

		// The BinaryAssetsDirectory is only required if you want to run the tests directly
		// without call the makefile target test. If not informed it will look for the
		// default path defined in controller-runtime which is /usr/local/kubebuilder/.
		// Note that you must have the required binaries setup under the bin directory to perform
		// the tests directly. When we run make test it will be setup and used automatically.
		BinaryAssetsDirectory: filepath.Join("..", "..", "bin", "k8s",
			fmt.Sprintf("1.29.0-%s-%s", runtime.GOOS, runtime.GOARCH)),
	}

	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = projctlv1beta1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	By("setting up the controller service account")

	ctx := context.Background()
	setupSystemNamespace(ctx, k8sClient)
	setupServiceAccount(ctx, k8sClient)
	saCfg := new(rest.Config)
	*saCfg = *cfg
	saCfg.Impersonate.UserName = "system:serviceaccount:system:controller-manager"
	saClient, err = client.New(saCfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(saClient).NotTo(BeNil())

	saCluster, err = cluster.New(saCfg)
	Expect(err).NotTo(HaveOccurred())
	Expect(saCluster).NotTo(BeNil())
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

func setupSystemNamespace(ctx context.Context, client client.Client) {
	ns := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "system"}}
	Expect(client.Create(ctx, &ns)).To(Or(
		Succeed(),
		MatchError(apierrors.IsAlreadyExists, "IsAlreadyExists"),
	))
}

func setupServiceAccount(ctx context.Context, client client.Client) {
	saFiles := []string{"role", "service_account", "role_binding"}
	for _, saFile := range saFiles {
		testhelpers.ApplyFile(
			ctx, client,
			filepath.Join("..", "..", "config", "rbac", fmt.Sprintf("%s.yaml", saFile)),
			"system",
		)
	}
}

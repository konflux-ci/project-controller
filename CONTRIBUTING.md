# Contributing to this project

## Automated testing with CRC

After installing CRC and starting a cluster, we can use the `kubectl` context 
already prepared for us to connect to the cluster.

    kubectl config use-context crc-admin

This should allow us to run commands like:

    kubectl cluster-info

The output should resemble the following:

    Kubernetes control plane is running at https://api.crc.testing:6443

    To further debug and diagnose cluster problems, use 'kubectl cluster-info dump'.

Now we can run our tests:

    USE_EXISTING_CLUSTER=true KEEP_TEST_NAMESPACES=true ginkgo run -v internal/controller

Since we have `KEEP_TEST_NAMESPACES` set to `true` we can find namespaces with 
names like `test-ns-1-1234` in the cluster once the tests are done and inspect
the obejcts that were created.

## Manual testing with CRC

Login to CRC as *kubeadmin*. The password would be displayed when bringing up
CRC.

    oc login -u kubeadmin https://api.crc.testing:6443

Add the Application and Controller CRDs to the cluster by cloning the
[application-api repository][api]. Then load the CRDs to the cluster:

    oc apply -f $PATH_TO_APPLICATION_API/config/crd/bases/appstudio.redhat.com_applications.yaml
    oc apply -f $PATH_TO_APPLICATION_API/config/crd/bases/appstudio.redhat.com_components.yaml

[api]: https://github.com/redhat-appstudio/application-api/

Create the `project-controller-system` namespace and go into it:

    oc create namespace project-controller-system
    oc project project-controller-system

Create an image stream for the controller image, then build and push it to the
cluster:

    oc create imagestream project-controller
    make docker-build \
        IMG=default-route-openshift-image-registry.apps-crc.testing/project-controller-system/project-controller
    docker push --tls-verify=false \
        default-route-openshift-image-registry.apps-crc.testing/project-controller-system/project-controller

Deploy the controller:

    make deploy \
        IMG=default-route-openshift-image-registry.apps-crc.testing/project-controller-system/project-controller

Start viewing the controller logs:

    oc logs -n project-controller-system -l control-plane=controller-manager -f

Create a namespace to test the controller with:

    oc create namespace testns
    oc project testns

Create a project, a template and a development stream:

    oc apply -f config/samples/projctl_v1beta1_project.yaml
    oc apply -f config/samples/projctl_v1beta1_projectdevelopmentstreamtemplate.yaml

Monitor the logs to see if the controller reconcile loop runs successfully.

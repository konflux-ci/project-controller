# Contributing to this project

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

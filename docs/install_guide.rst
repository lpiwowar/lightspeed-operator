Installation Guide
===================

This page covers prerequisites, installing the operator, setting up LLM
credentials, and deploying ``OpenStackLightspeed``. No cluster yet? See
:ref:`dont-have-a-cluster-yet-crc`

Prerequisites
-------------

* An OpenShift cluster (4.16+).

  .. warning::

     Known issue: the console UI does not currently work on OpenShift 4.20
     or newer. Stick to 4.18 until this is resolved.

* An LLM endpoint and API key — any provider from
  :ref:`supported-providers` works.
* A free Red Hat Developer account, to pull some images from
  ``registry.redhat.io`` — see :ref:`redhat-registry-access` below.
* Optional: RHOSO installed, only needed for the experimental
  cluster-introspection feature (:doc:`usage`).

.. _redhat-registry-access:

Access to registry.redhat.io images
------------------------------------

The console plugin and OKP images (both always deployed) come from
``registry.redhat.io`` rather than ``quay.io``. This requires a **free**
account you can create by following these steps:

#. Create a free account at `developers.redhat.com
   <https://developers.redhat.com/>`_.
#. Download a pull secret from the `Hybrid Cloud Console
   <https://console.redhat.com/>`_.
#. Add it to your cluster:

   * **CRC**: pass it as ``PULL_SECRET`` when creating the cluster — see
     :ref:`dont-have-a-cluster-yet-crc`.
   * **Existing cluster**: merge it into the cluster-wide pull secret:

     .. code-block:: bash

        oc get secret/pull-secret -n openshift-config -o jsonpath='{.data.\.dockerconfigjson}' \
          | base64 -d > pull-secret.json
        # merge the downloaded auths into pull-secret.json, then:
        oc set data secret/pull-secret -n openshift-config \
          --from-file=.dockerconfigjson=pull-secret.json

#. Verify the cluster itself can pull, using its own pull secret:

   .. code-block:: console

      $ oc run registry-pull-test --image=registry.redhat.io/openshift-lightspeed/lightspeed-console-plugin-pf5-rhel9:1.0.12 --restart=Never
      $ oc get pod registry-pull-test

   Any status other than ``ImagePullBackOff``/``ErrImagePull`` means it's
   working — clean up with ``oc delete pod registry-pull-test``. If you do
   see it, the secret from the previous step didn't propagate — see
   :ref:`console-widget-not-appearing`.

.. _installing-the-operator:

Installing the operator
------------------------

#. **Operators → OperatorHub**, search for **"OpenStack Lightspeed
   (Community)"**.
#. Click **Install**, choosing the ``openstack-lightspeed`` namespace.
#. Track progress under **Operators → Installed Operators**, or:

   .. code-block:: console

      $ oc get -n openstack-lightspeed pods
      NAME                                                              READY   STATUS    RESTARTS   AGE
      openstack-lightspeed-operator-controller-manager-76df7fbfb5wggr   1/1     Running   0          72s

.. note::

   Currently published for OpenShift 4.16 and 4.18 specifically — on
   other versions it won't appear in OperatorHub search. Use the
   alternative below instead.

**Alternative — deploy from source** (for testing an unreleased build, or
if your OpenShift version isn't in the catalog yet):

.. code-block:: bash

   git clone https://github.com/openstack-k8s-operators/lightspeed-operator.git
   cd lightspeed-operator
   make openstack-lightspeed-deploy

This sets up its own ``CatalogSource``, namespace, and ``Subscription`` —
bypassing OperatorHub entirely.

Setting up LLM credentials
----------------------------

You need an API key, endpoint URL, and model name.

Create the API key secret — the key **must** be named ``apitoken``:

.. code-block:: bash

   oc apply -f - <<EOF
   apiVersion: v1
   kind: Secret
   type: Opaque
   metadata:
     name: openstack-lightspeed-apitoken
     namespace: openstack-lightspeed
   stringData:
     apitoken: <your-llm-api-key>
   EOF

Using a self-hosted endpoint with a self-signed certificate (e.g. vLLM,
Ollama)? Add its CA bundle too — any key name works, PEM data is all
that's parsed:

.. code-block:: bash

   oc apply -f - <<EOF
   apiVersion: v1
   kind: ConfigMap
   metadata:
     name: openstack-lightspeed-certs
     namespace: openstack-lightspeed
   data:
     cert: |
   $(sed 's/^/    /' /path/to/cert.crt)
   EOF

Public providers (Gemini, OpenAI, etc.) don't need this — skip straight to
the next step.

Deploying OpenStackLightspeed
--------------------------------

At minimum, set ``llmEndpoint``, ``llmEndpointType``, ``modelName``, and
``llmCredentials`` — see :doc:`configuration` for the full list of
supported ``llmEndpointType`` values and everything else:

.. code-block:: yaml

   apiVersion: lightspeed.openstack.org/v1beta1
   kind: OpenStackLightspeed
   metadata:
     name: openstack-lightspeed
     namespace: openstack-lightspeed
   spec:
     llmEndpoint: https://<llm-provider-host>:<port>/v1
     llmEndpointType: <provider-type>
     llmCredentials: openstack-lightspeed-apitoken
     modelName: <model-name>
     tlsCACertBundle: openstack-lightspeed-certs # optional

This deploys the full stack: the AI engine (lightspeed-stack and
llama-stack/OGX), PostgreSQL, OKP, and the console plugin.

Verifying the deployment
---------------------------

.. code-block:: bash

   oc describe -n openstack-lightspeed openstacklightspeed
   oc get -n openstack-lightspeed deployments,pods

Not reaching ``Ready``? See :doc:`troubleshooting`.

Accessing the assistant
---------------------------

.. code-block:: bash

   oc whoami --show-console

Open that URL and use the Lightspeed widget (lower-right corner). First
time activating the plugin, you may need to click **refresh** on the
console notification that appears.

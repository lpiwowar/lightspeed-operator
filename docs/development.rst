Development
============

This page is for contributors and anyone testing changes locally — not
needed if you're just installing and using the operator.

.. _dont-have-a-cluster-yet-crc:

Local cluster (CRC)
-----------------------

For local development/testing only (not for trying the assistant for real
— CRC is resource-constrained). Deploy a CRC cluster before
:ref:`installing-the-operator`:

.. code-block:: bash

   git clone https://github.com/openstack-k8s-operators/install_yamls.git
   cd install_yamls/devsetup
   make download_tools

   CRC_VERSION=2.51.0 PULL_SECRET=~/work/pull-secret CRC_MONITORING_ENABLED=true CPUS=12 MEMORY=25600 DISK=100 make crc
   make crc_attach_default_interface
   eval $(crc oc-env)
   cd ../..

``PULL_SECRET`` is the same pull secret from :ref:`redhat-registry-access`.

CRC's console is always at a fixed address:
`console-openshift-console.apps-crc.testing
<https://console-openshift-console.apps-crc.testing>`_ — not something you
look up with ``oc whoami --show-console``.

Running CRC remotely? Reach that console with ``sshuttle``:

* Add to your local ``/etc/hosts`` (keep the IP as-is):
  ``192.168.130.11 api.crc.testing canary-openshift-ingress-canary.apps-crc.testing console-openshift-console.apps-crc.testing default-route-openshift-image-registry.apps-crc.testing downloads-openshift-console.apps-crc.testing oauth-openshift.apps-crc.testing``
* Run ``sshuttle -r $remote_username@$remote_server 192.168.130.0/24``.

Architecture
---------------

.. mermaid::

   graph TB
       User[System Administrator] -->|uses console widget| Plugin

       subgraph ns["openstack-lightspeed namespace"]
           CR[OpenStackLightspeed CR] --> Operator[lightspeed-operator]
           Operator --> Plugin[Console Plugin]
           Operator --> DB[(PostgreSQL)]
           Operator --> OKP[OKP]
           Operator --> Pod

           subgraph Pod["lightspeed-stack pod"]
               API[lightspeed-service-api] --> OGX[llama-stack]
               OGX -.-> MCP[MCP tools sidecar]
           end
       end

       Plugin --> API
       OGX --> OKP
       OGX --> LLM[Your LLM endpoint]
       MCP -.->|read-only, optional| OSP[Your OpenStack / OpenShift APIs]

**OKP is deployed on every install, not opt-in.** It's the default RAG
source; the bundled community documentation is available too, but only if
you explicitly opt in. See :doc:`configuration` for details.

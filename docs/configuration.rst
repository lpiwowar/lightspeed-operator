Configuration
=============

Everything is configured through the ``OpenStackLightspeed`` custom
resource (``lightspeed.openstack.org/v1beta1``). This page documents every
field in its ``spec``.

Core fields
-----------

.. list-table::
   :header-rows: 1
   :widths: 20 10 70

   * - Field
     - Required
     - Description
   * - ``llmEndpoint``
     - Yes
     - URL of the LLM endpoint (e.g. ``https://api.openai.com/v1``). Must
       start with ``http://`` or ``https://``.
   * - ``llmEndpointType``
     - Yes
     - Provider type. See :ref:`supported-providers`.
   * - ``modelName``
     - Yes
     - Model name to use at ``llmEndpoint``.
   * - ``llmCredentials``
     - Yes
     - ``Secret`` name (same namespace) with the API token under key
       ``apitoken``.
   * - ``tlsCACertBundle``
     - No
     - ``ConfigMap`` name (same namespace) with a CA bundle for the LLM endpoint.
   * - ``maxTokensForResponse``
     - No
     - Max response tokens. Minimum ``1``. Defaults to ``2048``.
   * - ``llmProjectID``
     - No
     - Required by some providers (e.g. WatsonX).
   * - ``llmDeploymentName``
     - No
     - Required by some providers (e.g. Azure OpenAI).
   * - ``llmAPIVersion``
     - No
     - Required by some providers (e.g. Azure OpenAI).

.. _supported-providers:

Supported LLM providers (``llmEndpointType``)
------------------------------------------------

* ``openai`` — OpenAI-compatible endpoints (Ollama, vLLM, etc.)
* ``azure_openai`` — Azure OpenAI (needs ``llmDeploymentName``, ``llmAPIVersion``)
* ``watsonx`` — IBM watsonx.ai (needs ``llmProjectID``)
* ``rhoai_vllm`` — vLLM via Red Hat OpenShift AI
* ``rhelai_vllm`` — vLLM via RHEL AI
* ``gemini`` — Google Gemini

.. tip::

   This list grows over time. Check
   ``oc explain openstacklightspeed.spec.llmEndpointType`` on your cluster
   for the current, authoritative list.

Logging
-----------------------

.. list-table::
   :header-rows: 1
   :widths: 25 15 60

   * - Field
     - Default
     - Description
   * - ``ogx.logLevel``
     - ``all=info``
     - OGX container. Standard level, or ``component=level`` pairs
       (e.g. ``core=debug,providers=info``).
   * - ``lcore.logLevel``
     - ``INFO``
     - lightspeed-service-api container. ``DEBUG``/``INFO``/``WARNING``/``ERROR``/``CRITICAL``.
   * - ``database.logLevel``
     - ``INFO``
     - PostgreSQL container. ``DEBUG`` also logs every SQL statement.

Dataverse exporter (``dataverseExporter``)
--------------------------------------------

.. list-table::
   :header-rows: 1
   :widths: 25 15 60

   * - Field
     - Default
     - Description
   * - ``dataverseExporter.logLevel``
     - ``INFO``
     - Feedback/transcript exporter sidecar. ``DEBUG``/``INFO``/``WARNING``/``ERROR``/``CRITICAL``.
   * - ``dataverseExporter.feedback.enabled``
     - ``true``
     - User feedback collection (thumbs-up/down on responses).
   * - ``dataverseExporter.transcripts.enabled``
     - ``false``
     - Full conversation transcript collection.

Persistent storage (``database``)
------------------------------------

PostgreSQL always gets a PersistentVolumeClaim — this field only overrides
its size/class, it doesn't control whether one exists:

.. code-block:: yaml

   spec:
     database:
       size: "5Gi"                # default: 1Gi
       class: "my-storage-class"  # default: cluster's default StorageClass

Container resources (``resources``)
--------------------------------------

Every container has a default request/limit. Setting one replaces its
default entirely:

.. code-block:: yaml

   spec:
     ogx:
       resources:
         requests: {cpu: "500m", memory: "2Gi"}
         limits: {cpu: "2", memory: "8Gi"}
     console:
       resources:
         requests: {cpu: "50m", memory: "64Mi"}
         limits: {cpu: "200m", memory: "256Mi"}
     lcore:
       resources:
         requests: {cpu: "250m", memory: "512Mi"}
         limits: {cpu: "1", memory: "2Gi"}
     database:
       resources:
         requests: {cpu: "30m", memory: "300Mi"}
         limits: {cpu: "500m", memory: "2Gi"}
     okp:
       resources:
         requests: {cpu: "500m", memory: "2Gi"}
         limits: {cpu: "2", memory: "4Gi"}

.. _offline-knowledge-portal:

Offline Knowledge Portal (``okp``)
--------------------------------------

.. important::

   OKP is deployed on **every** install — ``spec.okp`` configures it, it
   doesn't gate whether it's deployed. Pulling its image needs the same
   free ``registry.redhat.io`` account as :ref:`redhat-registry-access`.

.. code-block:: yaml

   spec:
     okp: {}   # no access key: browse individual pages, full-text search doesn't work

.. code-block:: yaml

   spec:
     okp:
       accessKey: okp-access-key-secret   # Secret key: "access_key"

* **No ``accessKey``** (default) — you can navigate directly to and read
  individual documentation and product lifecycle pages. The full-text
  search index, Solutions, and Articles are encrypted and require a key,
  so keyword search across the corpus doesn't work. What upstream users
  run on.
* **With ``accessKey``** — unlocks that search index plus the encrypted
  knowledgebase. Needs an active Red Hat Satellite subscription (`get one
  <https://access.redhat.com/offline/access>`_) — a bonus if you already
  have one, not something every user needs.

By default, **RAG grounding is OKP-only** — the bundled community
documentation is disabled unless you set ``dev.okpRagOnly: false`` (below).

.. _quota-enforcement:

Quota enforcement (``quotas``)
------------------------------

Configure one or more limiters to enable token quota enforcement. The
operator uses its managed PostgreSQL instance for quota storage. Omitting
``quotas`` or leaving ``limiters`` empty disables enforcement.

.. code-block:: yaml

   spec:
     quotas:
       limiters:
         - name: per-user-hourly
           type: userLimiter
           initialQuota: 1000
           quotaIncrease: 1000
           period: "1 hour"
         - name: cluster-daily
           type: clusterLimiter
           initialQuota: 100000
           quotaIncrease: 100000
           period: "1 day"
       scheduler:
         period: 10
       enableTokenHistory: true

Each entry in ``limiters`` requires these fields:

.. list-table::
   :header-rows: 1
   :widths: 25 75

   * - Field
     - Description
   * - ``name``
     - A human-readable limiter name.
   * - ``type``
     - ``userLimiter`` for a per-user quota, or ``clusterLimiter`` for one
       quota shared by the cluster.
   * - ``initialQuota``
     - Number of tokens granted when the limiter resets. Must be zero or
       greater.
   * - ``quotaIncrease``
     - Number of tokens added by the scheduler at each quota interval. Must
       be zero or greater.
   * - ``period``
     - Interval that controls when the limiter resets or increases, such as
       ``"30 seconds"``, ``"1 hour"``, ``"1 day"``, or
       ``"1 hour 30 minutes"``.

``scheduler`` is optional and configures the background process that checks
limiters for reset or increase and reconnects to the database after a
connection failure:

* ``period``: check interval in seconds. Default: ``5``.
* ``databaseReconnectionCount``: number of database reconnection attempts.
  Default: ``10``.
* ``databaseReconnectionDelay``: delay in seconds between reconnection
  attempts. Default: ``1``.

Set ``enableTokenHistory: true`` to record per-user, model, and provider token
usage for auditing. It does not affect enforcement and defaults to ``false``.

Developer / experimental options (``dev``)
-----------------------------------------------

.. warning::

   Not part of the stable API — may change without notice.

.. code-block:: yaml

   spec:
     dev:
       featureFlags:
         - rhoso_mcps   # enables the read-only MCP introspection sidecar
       okpChunkFilterQuery: "product:(*openstack* OR *openshift*)"  # example override
       okpRagOnly: true  # include bundled community docs too, not just OKP
       rhosMCP:
         resources:
           requests: {cpu: "50m", memory: "300Mi"}
           limits: {memory: "500Mi"}
         config: |
           debug: true
           workers: 4

* ``okpChunkFilterQuery`` and ``okpRagOnly`` take effect immediately, with
  no ``featureFlags`` entry needed — they're independent of
  ``rhoso_mcps``. If unset, ``okpChunkFilterQuery`` auto-detects your
  OpenShift/RHOSO versions instead of using the literal example above.
* ``rhoso_mcps`` — the one flag that does need to be set. Deploys the MCP
  introspection sidecar, which is read-only **by default**. See
  :doc:`usage`.
* ``rhosMCP`` is deep-merged on top of the operator's own defaults
  — ``config`` can override anything the default config sets, including the
  ``allow_write`` flags that keep introspection read-only. ``resources`` sets
  compute resources for the rhos-mcps sidecar (defaults shown above). Only set
  ``config`` if you understand exactly what you're overriding.

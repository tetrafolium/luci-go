# Copyright 2018 The LUCI Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

load('@stdlib//internal/graph.star', 'graph')
load('@stdlib//internal/lucicfg.star', 'lucicfg')
load('@stdlib//internal/validate.star', 'validate')

load('@stdlib//internal/luci/common.star', 'builder_ref', 'keys', 'triggerer')
load('@stdlib//internal/luci/lib/scheduler.star', 'schedulerimpl')
load('@stdlib//internal/luci/lib/swarming.star', 'swarming')


def _builder(
      ctx,
      *,

      name=None,
      bucket=None,
      recipe=None,

      # Execution environment parameters.
      properties=None,
      service_account=None,
      caches=None,
      execution_timeout=None,

      # Scheduling parameters.
      dimensions=None,
      priority=None,
      swarming_tags=None,
      expiration_timeout=None,

      # LUCI Scheduler parameters.
      schedule=None,
      triggering_policy=None,

      # Tweaks.
      build_numbers=None,
      experimental=None,
      task_template_canary_percentage=None,
      repo=None,

      # Deprecated stuff, candidate for deletion.
      luci_migration_host=None,

      # Relations.
      triggers=None,
      triggered_by=None,
      notifies=None
  ):
  """Defines a generic builder.

  It runs some recipe in some requested environment, passing it a struct with
  given properties. It is launched whenever something triggers it (a poller or
  some other builder, or maybe some external actor via Buildbucket or LUCI
  Scheduler APIs).

  The full unique builder name (as expected by Buildbucket RPC interface) is
  a pair `(<project>, <bucket>/<name>)`, but within a single project config
  this builder can be referred to either via its bucket-scoped name (i.e.
  `<bucket>/<name>`) or just via it's name alone (i.e. `<name>`), if this
  doesn't introduce ambiguities.

  The definition of what can *potentially* trigger what is defined through
  `triggers` and `triggered_by` fields. They specify how to prepare ACLs and
  other configuration of services that execute builds. If builder **A** is
  defined as "triggers builder **B**", it means all services should expect **A**
  builds to trigger **B** builds via LUCI Scheduler's EmitTriggers RPC or via
  Buildbucket's ScheduleBuild RPC, but the actual triggering is still the
  responsibility of **A**'s recipe.

  There's a caveat though: only Scheduler ACLs are auto-generated by the config
  generator when one builder triggers another, because each Scheduler job has
  its own ACL and we can precisely configure who's allowed to trigger this job.
  Buildbucket ACLs are left unchanged, since they apply to an entire bucket, and
  making a large scale change like that (without really knowing whether
  Buildbucket API will be used) is dangerous. If the recipe triggers other
  builds directly through Buildbucket, it is the responsibility of the config
  author (you) to correctly specify Buildbucket ACLs, for example by adding the
  corresponding service account to the bucket ACLs:

  ```python
  luci.bucket(
      ...
      acls = [
          ...
          acl.entry(acl.BUILDBUCKET_TRIGGERER, <builder service account>),
          ...
      ],
  )
  ```

  This is not necessary if the recipe uses Scheduler API instead of Buildbucket.

  Args:
    name: name of the builder, will show up in UIs and logs. Required.
    bucket: a bucket the builder is in, see luci.bucket(...) rule. Required.
    recipe: a recipe to run, see luci.recipe(...) rule. Required.

    properties: a dict with string keys and JSON-serializable values, defining
        properties to pass to the recipe. Supports the module-scoped defaults.
        They are merged (non-recursively) with the explicitly passed properties.
    service_account: an email of a service account to run the recipe under:
        the recipe (and various tools it calls, e.g. gsutil) will be able to
        make outbound HTTP calls that have an OAuth access token belonging to
        this service account (provided it is registered with LUCI). Supports
        the module-scoped default.
    caches: a list of swarming.cache(...) objects describing Swarming named
        caches that should be present on the bot. See swarming.cache(...) doc
        for more details. Supports the module-scoped defaults. They are joined
        with the explicitly passed caches.
    execution_timeout: how long to wait for a running build to finish before
        forcefully aborting it and marking the build as timed out. If None,
        defer the decision to Buildbucket service. Supports the module-scoped
        default.

    dimensions: a dict with swarming dimensions, indicating requirements for
        a bot to execute the build. Keys are strings (e.g. `os`), and values are
        either strings (e.g. `Linux`), swarming.dimension(...) objects (for
        defining expiring dimensions) or lists of thereof. Supports the
        module-scoped defaults. They are merged (non-recursively) with the
        explicitly passed dimensions.
    priority: int [1-255] or None, indicating swarming task priority, lower is
        more important. If None, defer the decision to Buildbucket service.
        Supports the module-scoped default.
    swarming_tags: a list of tags (`k:v` strings) to assign to the Swarming task
        that runs the builder. Each tag will also end up in `swarming_tag`
        Buildbucket tag, for example `swarming_tag:builder:release`. Supports
        the module-scoped defaults. They are joined with the explicitly passed
        tags.
    expiration_timeout: how long to wait for a build to be picked up by a
        matching bot (based on `dimensions`) before canceling the build and
        marking it as expired. If None, defer the decision to Buildbucket
        service. Supports the module-scoped default.

    schedule: string with a cron schedule that describes when to run this
        builder. See [Defining cron schedules](#schedules_doc) for the expected
        format of this field. If None, the builder will not be running
        periodically.
    triggering_policy: scheduler.policy(...) struct with a configuration that
        defines when and how LUCI Scheduler should launch new builds in response
        to triggering requests from luci.gitiles_poller(...) or from
        EmitTriggers API. Does not apply to builds started directly through
        Buildbucket. By default, only one concurrent build is allowed and while
        it runs, triggering requests accumulate in a queue. Once the build
        finishes, if the queue is not empty, a new build starts right away,
        "consuming" all pending requests. See scheduler.policy(...) doc for more
        details. Supports the module-scoped default.

    build_numbers: if True, generate monotonically increasing contiguous numbers
        for each build, unique within the builder. If None, defer the decision
        to Buildbucket service. Supports the module-scoped default.
    experimental: if True, by default a new build in this builder will be marked
        as experimental. This is seen from recipes and they may behave
        differently (e.g. avoiding any side-effects). If None, defer the
        decision to Buildbucket service. Supports the module-scoped default.
    task_template_canary_percentage: int [0-100] or None, indicating percentage
        of builds that should use a canary swarming task template. If None,
        defer the decision to Buildbucket service. Supports the module-scoped
        default.
    repo: URL of a primary git repository (starting with `https://`) associated
        with the builder, if known. It is in particular important when using
        luci.notifier(...) to let LUCI know what git history it should use to
        chronologically order builds on this builder. If unknown, builds will be
        ordered by creation time. If unset, will be taken from the configuration
        of luci.gitiles_poller(...) that trigger this builder if they all poll
        the same repo.

    luci_migration_host: deprecated setting that was important during the
        migration from Buildbot to LUCI. Refer to Buildbucket docs for the
        meaning. Supports the module-scoped default.

    triggers: builders this builder triggers.
    triggered_by: builders or pollers this builder is triggered by.
    notifies: list of luci.notifier(...) the builder notifies when it changes
        its status. This relation can also be defined via `notified_by` field in
        luci.notifier(...).
  """
  name = validate.string('name', name)
  bucket_key = keys.bucket(bucket)
  recipe_key = keys.recipe(recipe)

  # TODO(vadimsh): Validators here and in lucicfg.rule(..., defaults = ...) are
  # duplicated. There's probably a way to avoid this by introducing a Schema
  # object.
  props = {
      'name': name,
      'bucket': bucket_key.id,
      'properties': validate.str_dict('properties', properties),
      'service_account': validate.string('service_account', service_account, required=False),
      'caches': swarming.validate_caches('caches', caches),
      'execution_timeout': validate.duration('execution_timeout', execution_timeout, required=False),
      'dimensions': swarming.validate_dimensions('dimensions', dimensions, allow_none=True),
      'priority': validate.int('priority', priority, min=1, max=255, required=False),
      'swarming_tags': swarming.validate_tags('swarming_tags', swarming_tags),
      'expiration_timeout': validate.duration('expiration_timeout', expiration_timeout, required=False),
      'schedule': validate.string('schedule', schedule, required=False),
      'triggering_policy': schedulerimpl.validate_policy('triggering_policy', triggering_policy, required=False),
      'build_numbers': validate.bool('build_numbers', build_numbers, required=False),
      'experimental': validate.bool('experimental', experimental, required=False),
      'task_template_canary_percentage': validate.int('task_template_canary_percentage', task_template_canary_percentage, min=0, max=100, required=False),
      'repo': validate.repo_url('repo', repo, required=False),
      'luci_migration_host': validate.string('luci_migration_host', luci_migration_host, allow_empty=True, required=False)
  }

  # Merge explicitly passed properties with the module-scoped defaults.
  for k, prop_val in props.items():
    var = getattr(ctx.defaults, k, None)
    def_val = var.get() if var else None
    if def_val == None:
      continue
    if k in ('properties', 'dimensions'):
      props[k] = _merge_dicts(def_val, prop_val)
    elif k in ('caches', 'swarming_tags'):
      props[k] = _merge_lists(def_val, prop_val)
    elif prop_val == None:
      props[k] = def_val

  # Add a node that carries the full definition of the builder.
  builder_key = keys.builder(bucket_key.id, name)
  graph.add_node(builder_key, props = props)
  graph.add_edge(bucket_key, builder_key)
  graph.add_edge(builder_key, recipe_key)

  # Allow this builder to be referenced from other nodes via its bucket-scoped
  # name and via a global (perhaps ambiguous) name. See builder_ref.add(...).
  # Ambiguity is checked during the graph traversal via builder_ref.follow(...).
  builder_ref_key = builder_ref.add(builder_key)

  # Setup nodes that indicate this builder can be referenced in 'triggered_by'
  # relations (either via its bucket-scoped name or via its global name).
  triggerer_key = triggerer.add(builder_key)

  # Link to builders triggered by this builder.
  for t in validate.list('triggers', triggers):
    graph.add_edge(
        parent = triggerer_key,
        child = keys.builder_ref(t),
        title = 'triggers',
    )

  # And link to nodes this builder is triggered by.
  for t in validate.list('triggered_by', triggered_by):
    graph.add_edge(
        parent = keys.triggerer(t),
        child = builder_ref_key,
        title = 'triggered_by',
    )

  # Subscribe notifiers to this builder.
  for n in validate.list('notifies', notifies):
    graph.add_edge(
        parent = keys.notifier(n),
        child = builder_ref_key,
        title = 'notifies',
    )

  return graph.keyset(builder_key, builder_ref_key, triggerer_key)


def _merge_dicts(defaults, extra):
  out = dict(defaults.items())
  for k, v in extra.items():
    if v != None:
      out[k] = v
  return out


def _merge_lists(defaults, extra):
  return defaults + extra


builder = lucicfg.rule(
    impl = _builder,
    defaults = validate.vars_with_validators({
        'properties': validate.str_dict,
        'service_account': validate.string,
        'caches': swarming.validate_caches,
        'execution_timeout': validate.duration,
        'dimensions': swarming.validate_dimensions,
        'priority': lambda attr, val: validate.int(attr, val, min=1, max=255),
        'swarming_tags': swarming.validate_tags,
        'expiration_timeout': validate.duration,
        'triggering_policy': schedulerimpl.validate_policy,
        'build_numbers': validate.bool,
        'experimental': validate.bool,
        'task_template_canary_percentage': lambda attr, val: validate.int(attr, val, min=1, max=100),
        'luci_migration_host': validate.string,
    }),
)

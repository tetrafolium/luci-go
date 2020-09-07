-- Copyright 2019 The LUCI Authors.
--
-- Licensed under the Apache License, Version 2.0 (the "License");
-- you may not use this file except in compliance with the License.
-- You may obtain a copy of the License at
--
--      http://www.apache.org/licenses/LICENSE-2.0
--
-- Unless required by applicable law or agreed to in writing, software
-- distributed under the License is distributed on an "AS IS" BASIS,
-- WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
-- See the License for the specific language governing permissions and
-- limitations under the License.

--------------------------------------------------------------------------------
-- This script initializes a ResultDB Spanner database.

-- Stores the invocations.
-- This is the root table for much of the other data and tables, which define the
-- hierarchy (dependency graph, subsets of interest) for invocations.
CREATE TABLE Invocations (
  -- Identifies an invocation.
  -- Format: "${hex(sha256(user_provided_id)[:8])}:${user_provided_id}".
  InvocationId STRING(MAX) NOT NULL,

  -- A random value in [0, Shards) where Shards constant is
  -- defined in code.
  -- Used in global secondary indexes, to prevent hot spots.
  -- The maximum value of ShardId in Spanner can be determined by querying the
  -- very first row in InvocationsByExpiration index.
  ShardId INT64 NOT NULL,

  -- Invocation state, see InvocationState in invocation.proto
  State INT64 NOT NULL,

  -- Security realm this invocation belongs to.
  -- Used to enforce ACLs.
  Realm STRING(64) NOT NULL,

  -- When to delete the invocation from the table.
  InvocationExpirationTime TIMESTAMP NOT NULL,

  -- When to delete expected test results from this invocation.
  -- When expected results are removed, this column is set to NULL.
  ExpectedTestResultsExpirationTime TIMESTAMP,

  -- When the invocation was created.
  CreateTime TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),

  -- LUCI identity who created the invocation, typically "user:<email>".
  CreatedBy STRING(MAX),

  -- When the invocation was finalized.
  FinalizeTime TIMESTAMP OPTIONS (allow_commit_timestamp=true),

  -- When to force invocation finalization.
  Deadline TIMESTAMP NOT NULL,

  -- List of colon-separated key-value tags.
  -- Corresponds to Invocation.tags in invocation.proto.
  Tags ARRAY<STRING(MAX)>,

  -- Value of CreateInvocationRequest.request_id.
  -- Used to dedup invocation creation requests.
  CreateRequestId STRING(MAX),

  -- Requests to export the invocation to BigQuery, see also
  -- Invocation.bigquery_exports in invocation.proto.
  -- Each array element is a binary-encoded luci.resultdb.v1.BigQueryExport
  -- message.
  BigQueryExports ARRAY<BYTES(MAX)>,

  -- Value of Invocation.producer_resource. See its documentation.
  ProducerResource STRING(MAX),

  -- Counter of TesultResults that belongs to this invocation directly.
  TestResultCount INT64,

  -- If this invocation is to be queried (e.g. for test results history) by an
  -- ordinal range, such as a commit range, set the following two fields for
  -- indexing.
  -- Either _both_ Ordinal and OrdinalDomain need to be NOT NULL to be indexed
  -- by ordinal, or _both_ are expected to be NULL to skip this index.

  -- A numeric value, where higher values are more recent.
  Ordinal INT64,

  -- A string, e.g. 'gitiles://<host>/<project>/<ref>', that provides context
  -- for the Ordinal column, e.g. if it is to be treated as a commit position.
  OrdinalDomain STRING(MAX),

  -- If this invocation is to be queried by a time range, e.g. for test results
  -- history query, set this field for indexing. If set, this should match
  -- CreateTime.
  -- Nullable to skip indexing some invocations.
  HistoryTime TIMESTAMP OPTIONS (allow_commit_timestamp=true),

) PRIMARY KEY (InvocationId);

-- Used by test results history to find a history of test results ordered by
-- invocation timestamp.
CREATE NULL_FILTERED INDEX InvocationsByTimestamp
  ON Invocations (Realm, HistoryTime DESC);

-- Used by test results history, to find test results ordered by e.g. commit
-- position.
CREATE NULL_FILTERED INDEX InvocationsByOrdinal
  ON Invocations (Realm, OrdinalDomain, Ordinal DESC);

-- Index of invocations by expiration time.
-- Used by a cron job that periodically removes expired invocations.
CREATE INDEX InvocationsByInvocationExpiration
  ON Invocations (ShardId DESC, InvocationExpirationTime, InvocationId);

-- Index of invocations by expected test result expiration.
-- Used by a cron job that periodically removes expected test results.
CREATE NULL_FILTERED INDEX InvocationsByExpectedTestResultsExpiration
  ON Invocations (ShardId DESC, ExpectedTestResultsExpirationTime, InvocationId);

-- Stores ids of invocations included in another invocation.
-- Interleaved in Invocations table.
CREATE TABLE IncludedInvocations (
  -- ID of the including invocation, the "source" node of the edge.
  InvocationId STRING(MAX) NOT NULL,

  -- ID of the included invocation, the "target" node of the edge.
  IncludedInvocationId STRING(MAX) NOT NULL
) PRIMARY KEY (InvocationId, IncludedInvocationId),
  INTERLEAVE IN PARENT Invocations ON DELETE CASCADE;

-- Reverse of IncludedInvocations.
-- Used to find invocations including a given one.
CREATE INDEX ReversedIncludedInvocations
  ON IncludedInvocations (IncludedInvocationId, InvocationId);

-- Stores test results. Interleaved in Invocations.
CREATE TABLE TestResults (
  -- ID of the parent Invocations row.
  InvocationId STRING(MAX) NOT NULL,

  -- Unique identifier of the test,
  -- see also TestResult.test_id in test_result.proto.
  TestId STRING(MAX) NOT NULL,

  -- A suffix for PK to allow multiple test results for the same test id in
  -- a given invocation.
  -- Generated on the server.
  ResultId STRING(MAX) NOT NULL,

  -- key:value pairs in the test variant.
  -- See also TestResult.variant in test_result.proto.
  Variant ARRAY<STRING(MAX)>,

  -- A hex-encoded sha256 of concatenated "<key>:<value>\n" variant pairs.
  -- Used to filter test results by variant.
  VariantHash STRING(64) NOT NULL,

  -- Last time this row was modified.
  -- Given that we only create and delete row, for an existing row this equals
  -- row creation time.
  CommitTimestamp TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),

  -- Whether the test status was unexpected
  -- MUST be either NULL or True, to keep null-filtered index below thin.
  IsUnexpected BOOL,

  -- Test status, see TestStatus in test_result.proto.
  Status INT64 NOT NULL,

  -- Compressed summary of the test result for humans, in HTML.
  -- See span.Compressed type for details of compression.
  SummaryHTML BYTES(MAX),

  -- When the test execution started.
  StartTime TIMESTAMP,

  -- How long the test execution took, in microseconds.
  RunDurationUsec INT64,

  -- Tags associated with the test result, for example GTest-specific test
  -- status.
  Tags ARRAY<STRING(MAX)>,

  -- Name of the test file.
  -- See also TestResult.test_location.file_name field.
  TestLocationFileName STRING(MAX),

  -- Line number in the test file.
  -- See also TestResult.test_location.line field.
  TestLocationLine INT64,

) PRIMARY KEY (InvocationId, TestId, ResultId),
  INTERLEAVE IN PARENT Invocations ON DELETE CASCADE;

-- Stores artifacts. Interleaved in Invocations.
CREATE TABLE Artifacts (
  -- Id of the parent Invocations row.
  InvocationId STRING(MAX) NOT NULL,

  -- An invocation-local ID of the Artifact parent:
  -- *   "" for invocation-level artifacts.
  -- *   "tr/{test_id}/{result_id}" for test-result-level artifacts.
  --     test_id is NOT URL-encoded because result_id cannot have a slash.
  ParentId STRING(MAX) NOT NULL,

  -- Unique identifier of the artifact within the parent.
  -- May have slashes.
  -- Example: "stdout" of a test result.
  ArtifactId STRING(MAX) NOT NULL,

  -- Media type of the artifact content.
  ContentType STRING(MAX),

  -- Content size in bytes.
  Size INT64,

  -- Hash of the artifact content if it is stored in RBE-CAS.
  -- Format: "sha256:{hash}" where the hash is a lower-case hex-encoded SHA256
  -- hash of the artifact content.
  -- Example: e.g. "sha256:deadbeefdeadbeefdeadbeefdeadbeefdeadbeef"
  --
  -- The RBE-CAS instance is in the same Cloud project, named "artifacts".
  RBECASHash STRING(MAX),

  -- A string of format "isolate://{isolateServerHost}/{namespace}/{hash}"
  -- if this artifact is stored in isolate.
  -- TODO(nodir): remove this when we completely switch to ResultSink.
  IsolateURL STRING(MAX),
) PRIMARY KEY (InvocationId, ParentId, ArtifactId),
  INTERLEAVE IN PARENT Invocations ON DELETE CASCADE;

-- Unexpected test results for each invocation.
-- It is significantly smaller (<2%) than TestResult table and should be used
-- for most queries.
-- It includes TestId to be able to find all unexpected test result with a
-- given test id or a test id prefix.
CREATE NULL_FILTERED INDEX UnexpectedTestResults
  ON TestResults (InvocationId, TestId, IsUnexpected) STORING (VariantHash, Variant),
  INTERLEAVE IN Invocations;


-- Stores test exonerations, see TestExoneration in test_result.proto
CREATE TABLE TestExonerations (
  -- ID of the parent Invocations row.
  InvocationId STRING(MAX) NOT NULL,

  -- The exoneration applies only to test results with this exact test id.
  -- This is a foreign key to TestResults.TestId column.
  TestId STRING(MAX) NOT NULL,

  -- Server-generated exoneration ID.
  -- Uniquely identifies a test exoneration within an invocation.
  --
  -- Starts with "{hex(sha256(join(sorted('{p}\n' for p in Variant))))}:".
  -- The prefix can be used to reduce scanning for test exonerations for a
  -- particular test variant.
  ExonerationId STRING(MAX) NOT NULL,

  -- The exoneration applies only to test results with this exact test variant.
  Variant ARRAY<STRING(MAX)> NOT NULL,

  -- A hex-encoded sha256 of concatenated "<key>:<value>\n" variant pairs.
  -- Used in conjunction with TestResults.VariantHash column.
  VariantHash STRING(64) NOT NULL,

  -- Compressed explanation of the exoneration for humans, in HTML.
  -- See span.Compress type for details of compression.
  ExplanationHTML BYTES(MAX)
) PRIMARY KEY (InvocationId, TestId, ExonerationId),
  INTERLEAVE IN PARENT Invocations ON DELETE CASCADE;

-- Stores tasks to perform on invocations.
-- E.g. to export an invocation to a BigQuery table.
CREATE TABLE InvocationTasks (
  -- Type of the task. See "taskType" type in the Go code for examples.
  TaskType STRING(16) NOT NULL,

  -- Id of the task.
  TaskId STRING(MAX) NOT NULL,

  -- ID of the invocation to process.
  InvocationId STRING(MAX) NOT NULL,

  -- Depends on task type. See "taskType" type in the Go code for examples.
  Payload BYTES(MAX),

  -- When the task was created.
  CreateTime TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),

  -- When to process the task.
  -- ProcessAfter can be set to NOW indicating the invocation can be processed
  -- or a future time indicating the invocation is not available to process yet.
  -- ProcessAfter can be reset to a future time by a worker when it starts to
  -- work on this task to prevent other workers picking up the same one.
  ProcessAfter TIMESTAMP
) PRIMARY KEY (TaskType, TaskId);

-- Stores transactional tasks reminders.
-- See https://github.com/tetrafolium/luci-go/server/tq. Scanned by tq-sweeper-spanner.
CREATE TABLE TQReminders (
  ID STRING(MAX) NOT NULL,
  FreshUntil TIMESTAMP NOT NULL,
  Payload BYTES(102400) NOT NULL,
) PRIMARY KEY (ID ASC);

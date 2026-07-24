# Changelog

## To be Released

## v1.6.0

* feat(opentelemetry) Add ability to instrument execution of cron job through OpenTelemetry

## v1.5.0

* feat(cronsetup): add local mode

## v1.4.0

* feat(cronsetup): add option to provide etcd client
* refactor(cronsetup): replace `github.com/Scalingo/go-etcd-cron` with `cronsetup/internal/cron`

## v1.3.0

* feat(request_id) Inject `request_id` in context for each cron execution

## v1.2.1

* chore(go): corrective bump - Go version regression from 1.24.3 to 1.24

## v1.2.0

* chore(go): upgrade to Go 1.24

## v1.1.4

* Various dependencies updates

## v1.1.3

* fix(cronsetup): add missing err check when adding job [#394](https://github.com/Scalingo/go-utils/pull/394)
* build(deps): bump github.com/Scalingo/go-utils/logger from 1.1.1 to 1.2.0
* build(deps): bump go.etcd.io/etcd/client/v3 from 3.5.4 to 3.5.5

## v1.1.2

* chore(go): use go 1.17
* build(deps): bump go.etcd.io/etcd/client/v3 from 3.5.0 to 3.5.4

## v1.1.1

* Bump github.com/go-utils/logger from v1.0.0 to v1.1.0

## v1.1.0

* Bump Scalingo/go-etcd-cron to 1.3.0 and bump etcd client to 3.5.0
  [#207](https://github.com/Scalingo/go-utils/pull/207)
* Bump go version to 1.16

## v1.0.0, v1.0.1, v1.0.2

* Initial breakdown of go-utils into subpackages

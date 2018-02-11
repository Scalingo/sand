# Changelog

## v1.1.2

### global

* No more reference to go-internal-tools

## v1.1.1

### nsqconsumer

* Create nsqconsumer.Error to handle when no retry should be done

## v1.1.0

### errors

* New errors.Notef and errors.Wrapf with context to let the error handling system
  read the context and its logger

## v1.0.2

### logger

* New API of logrus removed `logger.AddHook(hook)`, it is now `logger.Hooks.Add(hook)`

## v1.0.1

### logger

* Fix plugin system, use pointer instead of value for manager

## v1.0.0

### mongo (api change)

* Add a logger in initialization

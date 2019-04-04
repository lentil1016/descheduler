[![CircleCI](https://circleci.com/gh/lentil1016/descheduler.svg?style=svg)](https://circleci.com/gh/lentil1016/descheduler)

# descheduler
A descheduler server for kubernetes cluster. 

This scheduler runs as a server, and it makes evicting decisions more "gentlely". 

Every time it assess resource status of the cluster and evicts certain number of pods(defined by `spec.rules.maxEvictSize` in config file). Then it waits for replica sets pods of which is evicted rebounded to fully ready. Then it reassess the cluster and do another evicting again.

It says you can get the nutrition you need from either food or pills, and I believe [kubernetes-incubator/descheduler](https://github.com/kubernetes-incubator/descheduler) is the pills, this project is the food.

## Feature

- Run as a server, not a job.
- Triggered deschedule by node ready event or by timer.
- Config node selector to limit the nodes descheduler will affect.
- Be able to deschedule:
  - the pods that can find prefered node
  - the pods with peer pods(pods created by the same SeplicaSet) on the same node
  - the pods with peer pods in cluster

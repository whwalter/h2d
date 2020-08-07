## h2d - helm2 detector 

*h2d* - detects tiller services and helm2 releases accross all namespaces in a cluster. If tiller is detected and there are no helm 2 releases, tiller is removed. Only the configMap storage driver is supported.

### Usage

	Remove tiller if there are no v2 releases in a namespace.

	Usage:
	  h2d [flags]

	Flags:
	  -h, --help                     help for h2d
	      --label-selector string    tiller labels (default "name=tiller,app=helm")
	      --remove-service-account   (false) remove tiller serviceaccount from empty namespaces
	      --remove-tiller            (false) remove tiller from empty namespaces
	      --service-account string   tiller serviceaccount name, required for serviceaccount removal


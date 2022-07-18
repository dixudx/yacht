package utils

import (
	"fmt"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// NewBasicKubeConfig creates a basic KubeConfig object
func NewBasicKubeConfig(serverURL, clusterName, userName string, caCert []byte) *clientcmdapi.Config {
	// Use the cluster and the username as the context name
	contextName := fmt.Sprintf("%s@%s", userName, clusterName)

	var insecureSkipTLSVerify bool
	if caCert == nil {
		insecureSkipTLSVerify = true
	}

	return &clientcmdapi.Config{
		Clusters: map[string]*clientcmdapi.Cluster{
			clusterName: {
				Server:                   serverURL,
				InsecureSkipTLSVerify:    insecureSkipTLSVerify,
				CertificateAuthorityData: caCert,
			},
		},
		Contexts: map[string]*clientcmdapi.Context{
			contextName: {
				Cluster:  clusterName,
				AuthInfo: userName,
			},
		},
		AuthInfos:      map[string]*clientcmdapi.AuthInfo{},
		CurrentContext: contextName,
	}
}

// NewKubeConfigWithToken creates a KubeConfig object with access to the API server using a token
func NewKubeConfigWithToken(serverURL, token string, caCert []byte) *clientcmdapi.Config {
	userName := "yacht"
	clusterName := "yacht-cluster"
	config := NewBasicKubeConfig(serverURL, clusterName, userName, caCert)
	config.AuthInfos[userName] = &clientcmdapi.AuthInfo{
		Token: token,
	}
	return config
}

// NewKubeConfigWithCertificates creates a KubeConfig object with access to the API server using tls certificates
func NewKubeConfigWithCertificates(serverURL string, caCert, certificateData, keyData []byte) *clientcmdapi.Config {
	userName := "yacht"
	clusterName := "yacht-cluster"
	config := NewBasicKubeConfig(serverURL, clusterName, userName, caCert)
	config.AuthInfos[userName] = &clientcmdapi.AuthInfo{
		ClientCertificateData: certificateData,
		ClientKeyData:         keyData,
	}
	return config
}

// LoadsKubeConfig tries to load kubeconfig from specified config file or in-cluster config
func LoadsKubeConfig(configFile string) (*rest.Config, error) {
	if len(configFile) == 0 {
		// use in-cluster config
		return rest.InClusterConfig()
	}

	clientConfig, err := clientcmd.LoadFromFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("error while loading kubeconfig from file %v: %v", configFile, err)
	}
	return clientcmd.NewDefaultClientConfig(*clientConfig, &clientcmd.ConfigOverrides{}).ClientConfig()
}

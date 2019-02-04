package main

import (
	"fmt"
	"strings"
	"path"

	vault "github.com/hashicorp/vault/api"
)

type KvClient struct {
	Client *vault.Client
}


func NewKvClient(cfg *vault.Config, token string) (*KvClient, error) {
	c, err := vault.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	c.SetToken(token)
	c.Auth()

	return &KvClient{Client: c}, nil
}

func (c *KvClient) Put(path string, data map[string]interface{}) (*vault.Secret, error) {

	path, err := c.composePath(path, "data")
	if err != nil {
		return nil, fmt.Errorf("Unable to determine path of secret: %q", err)
	}

	tmp := data
	data = make(map[string]interface{})
	data["data"] = tmp

	sec, err:= c.Client.Logical().Write(path, data)
	if err != nil {
		return nil, err
	}
	return sec, nil
}


func (c *KvClient) Get(path string) (map[string]interface{}, error) {

	path, err := c.composePath(path, "data")
	if err != nil {
		return nil, fmt.Errorf("Unable to determine path of secret: %q", err)
	}

	sec, err:= c.Client.Logical().Read(path)
	if err != nil {
		return nil, err
	}

	if sec == nil {
		return nil, fmt.Errorf("Secret does not exist")
	}

	return sec.Data["data"].(map[string]interface{}), nil
}


func (c *KvClient) Delete(path string) (*vault.Secret, error) {

	path, err := c.composePath(path, "data")
	if err != nil {
		return nil, fmt.Errorf("Unable to determine path of secret: %q", err)
	}

	sec, err:= c.Client.Logical().Read(path)
	if err != nil {
		return nil, err
	}

	if sec == nil {
		return nil, fmt.Errorf("Secret does not exist")
	}

	return c.Client.Logical().Delete(path)
}

func (c *KvClient) List(path string) ([]string, error) {

	path = strings.TrimLeft(path, "/")
	if path == "" {
		mounts, err := kv.Client.Sys().ListMounts()
		if err != nil {
			return nil, err
		}

		var backends []string
		for x, i := range mounts {
			if i.Type == "kv" {
				backends = append(backends, x)
			}
		}
		return backends, nil
	}

	path, err := c.composePath(path, "metadata")
	if err != nil {
		return nil, fmt.Errorf("Unable to determine path of secret: %q", err)
	}

	secret, err := c.Client.Logical().List(path)
	if err != nil {
		return nil, err
	}

	if secret == nil {
		return nil, fmt.Errorf("Secret does not exist")
	}

	var children []string
	for _, path := range secret.Data {
		children = strings.Split(strings.Trim(fmt.Sprint(path), "[]"), " ")
	}

	return children, nil
}

func (c *KvClient) ListRecursively(path string) ([]string, error) {

	children, err := c.List(path)
	if err != nil {
		return nil, err
	}

	if len(children) == 0 {
		return []string{}, nil
	}


	var paths []string
	for _, child := range children {

		// TODO: Remove "metadata" from path
		child = fmt.Sprint(path, child)

		    if strings.HasSuffix(child, "/") {
			childs, err := c.ListRecursively(child)
			if err != nil {
				return nil, err
			}
			paths = append(paths, childs...)
		} else {
			paths = append(paths, child)
		}
	}

	return paths, nil
}


func (c *KvClient) composePath(p , prefix string) (string, error) {

	// We don't want to use a wrapping call here so save any custom value and
	// restore after
	currentWrappingLookupFunc := c.Client.CurrentWrappingLookupFunc()
	    c.Client.SetWrappingLookupFunc(nil)
	defer c.Client.SetWrappingLookupFunc(currentWrappingLookupFunc)

	r := c.Client.NewRequest("GET", "/v1/sys/internal/ui/mounts/" + p)
	resp, err := c.Client.RawRequest(r)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return "", err
	}

	secret, err := vault.ParseSecret(resp.Body)
	if err != nil {
		return "", err
	}

	var mountPath string
	if mountPathRaw, ok := secret.Data["path"]; ok {
		mountPath = mountPathRaw.(string)
	}


	var realPath string
	switch {
	case p == mountPath, p == strings.TrimSuffix(mountPath, "/"):
		realPath = path.Join(mountPath, prefix)
	default:
		p = strings.TrimPrefix(p, mountPath)
		realPath = path.Join(mountPath, prefix, p)
	}

	return realPath, nil
}
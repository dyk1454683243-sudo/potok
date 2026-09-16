package transport

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Vault struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	KDFSalt   []byte    `json:"kdf_salt"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

var ErrConflict = errors.New("manifest changed on the server since the last push - pull first, then push again")

func (c *Client) FetchVault(name string) (v Vault, found bool, err error) {
	response, err := c.RequestJSON(http.MethodGet, "/vaults/"+name, &v)
	if err != nil {
		return Vault{}, false, err
	}
	switch response.StatusCode {
	case http.StatusOK:
		return v, true, nil
	case http.StatusNotFound:
		return Vault{}, false, nil
	default:
		return Vault{}, false, fmt.Errorf("failed to look up vault %q: %s", name, response.Status)
	}
}

func (c *Client) GetOrCreateVault(name string) (Vault, error) {
	v, found, err := c.FetchVault(name)
	if err != nil {
		return Vault{}, err
	}
	if found {
		return v, nil
	}

	createResp, err := c.RequestJSONBody(http.MethodPost, "/vaults", struct {
		Name string `json:"name"`
	}{name}, nil)
	if err != nil {
		return Vault{}, err
	}
	if createResp.StatusCode != http.StatusCreated {
		return Vault{}, fmt.Errorf("failed to create vault %q on the server: %s", name, createResp.Status)
	}

	v, found, err = c.FetchVault(name)
	if err != nil {
		return Vault{}, err
	}
	if !found {
		return Vault{}, fmt.Errorf("vault %q was created but could not be found immediately after", name)
	}
	return v, nil
}

func (c *Client) UploadBlob(vaultName string, ciphertext []byte) (string, error) {
	sum := sha256.Sum256(ciphertext)
	blobID := hex.EncodeToString(sum[:])

	response, err := c.RequestBody(http.MethodPut, "/vaults/"+vaultName+"/blobs/"+blobID, "application/octet-stream", bytes.NewReader(ciphertext))
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated && response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%s", ReadMessage(response))
	}

	return blobID, nil
}

func (c *Client) DownloadBlob(vaultName, blobID string) ([]byte, error) {
	response, err := c.Request(http.MethodGet, "/vaults/"+vaultName+"/blobs/"+blobID)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, errors.New(ReadMessage(response))
	}

	return io.ReadAll(response.Body)
}

type manifestGetResponse struct {
	Generation int64  `json:"generation"`
	Ciphertext string `json:"ciphertext"`
}

type manifestPutRequest struct {
	ExpectedGeneration int64  `json:"expected_generation"`
	Ciphertext         string `json:"ciphertext"`
}

type manifestPutResponse struct {
	Generation int64 `json:"generation"`
}

func (c *Client) CurrentManifestGeneration(vaultName string) (int64, error) {
	var m manifestGetResponse
	response, err := c.RequestJSON(http.MethodGet, "/vaults/"+vaultName+"/manifest", &m)
	if err != nil {
		return 0, err
	}
	switch response.StatusCode {
	case http.StatusOK:
		return m.Generation, nil
	case http.StatusNotFound:
		return 0, nil
	default:
		return 0, fmt.Errorf("failed to check current manifest generation: %s", response.Status)
	}
}

func (c *Client) FetchManifest(vaultName string) (ciphertext []byte, found bool, err error) {
	var m manifestGetResponse
	response, err := c.RequestJSON(http.MethodGet, "/vaults/"+vaultName+"/manifest", &m)
	if err != nil {
		return nil, false, err
	}
	switch response.StatusCode {
	case http.StatusOK:
		ciphertext, err := base64.StdEncoding.DecodeString(m.Ciphertext)
		if err != nil {
			return nil, false, fmt.Errorf("server returned a non-base64 manifest ciphertext: %w", err)
		}
		return ciphertext, true, nil
	case http.StatusNotFound:
		return nil, false, nil
	default:
		return nil, false, fmt.Errorf("failed to fetch manifest: %s", response.Status)
	}
}

func (c *Client) PutManifest(vaultName string, expectedGeneration int64, ciphertext []byte) error {
	var out manifestPutResponse
	response, err := c.RequestJSONBody(http.MethodPut, "/vaults/"+vaultName+"/manifest", manifestPutRequest{
		ExpectedGeneration: expectedGeneration,
		Ciphertext:         base64.StdEncoding.EncodeToString(ciphertext),
	}, &out)
	if err != nil {
		return err
	}

	if response.StatusCode == http.StatusConflict {
		return ErrConflict
	}
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to push manifest: %s", response.Status)
	}

	return nil
}

package storage

import (
	"context"
	"errors"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage/v4"
	"log"
)

type ContainerService struct {
	client *armstorage.BlobContainersClient
}

func NewContainerService(c *armstorage.BlobContainersClient) *ContainerService {
	return &ContainerService{c}
}
func (s *ContainerService) Create(ctx context.Context, rg, acct, name string) error {
	_, e := s.client.Create(ctx, rg, acct, name, armstorage.BlobContainer{}, nil)
	return e
}
func (s *ContainerService) Get(ctx context.Context, rg, acct, name string) (armstorage.BlobContainer, error) {
	r, e := s.client.Get(ctx, rg, acct, name, nil)
	if e != nil {
		return armstorage.BlobContainer{}, e
	}
	return r.BlobContainer, nil
}
func (s *ContainerService) Exists(ctx context.Context, rg, acct, name string) (bool, error) {
	_, e := s.Get(ctx, rg, acct, name)
	if e == nil {
		return true, nil
	}
	var re *azcore.ResponseError
	if errors.As(e, &re) && re.StatusCode == 404 {
		return false, nil
	}
	return false, e
}
func (s *ContainerService) Delete(ctx context.Context, rg, acct, name string) error {
	_, e := s.client.Delete(ctx, rg, acct, name, nil)
	return e
}
func (s *ContainerService) List(ctx context.Context, rg, acct string) ([]*armstorage.ListContainerItem, error) {
	p := s.client.NewListPager(rg, acct, nil)
	var out []*armstorage.ListContainerItem
	for p.More() {
		r, e := p.NextPage(ctx)
		if e != nil {
			return nil, e
		}
		out = append(out, r.Value...)
	}
	return out, nil
}
func (s *ContainerService) Update(ctx context.Context, rg, acct, name string) error {

	// This acts exactly like `az storage container update`
	response, err := s.client.Update(
		ctx,
		rg,
		acct,
		name,
		armstorage.BlobContainer{
			ContainerProperties: &armstorage.ContainerProperties{
				// OPTION A: Set public access level
				// Choices: armstorage.PublicAccessBlob, armstorage.PublicAccessContainer, or armstorage.PublicAccessNone
				PublicAccess: to.Ptr(armstorage.PublicAccessBlob),

				// OPTION B: Update custom metadata if passed
				Metadata: map[string]*string{
					"environment": to.Ptr("production"),
					"managedBy":   to.Ptr("armstorage-go"),
				},
			},
		},
		nil, // Optional parameter overrides (UpdateOptions)
	)
	if err != nil {
		log.Fatalf("Failed to update container properties: %v", err)
	}

	fmt.Printf("Successfully updated container ID: %s\n", *response.BlobContainer.ID)

	//_, e := s.client.Update(ctx, rg, acct, name, armstorage.BlobContainerUpdate{Metadata: map[string]*string{"updated-by": ptr("azstoragecli")}}, nil)
	return err
}

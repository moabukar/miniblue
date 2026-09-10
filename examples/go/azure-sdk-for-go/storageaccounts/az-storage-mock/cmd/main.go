package main

import (
	"az-storage-mock/internal/storage"
	"context"
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

const subscriptionID = "00000000-0000-0000-0000-000000000000"

var resourceGroup, endpoint string

func addFlags(c *cobra.Command, name bool) {
	c.Flags().String("account-name", "", "Storage account name")
	if name {
		c.Flags().String("name", "", "Container name")
	}
}

func values(c *cobra.Command, name bool) (string, string, error) {
	a, e := c.Flags().GetString("account-name")
	if e != nil {
		return "", "", e
	}
	if a == "" {
		return "", "", fmt.Errorf("--account-name is required")
	}
	if !name {
		return "", a, nil
	}
	n, e := c.Flags().GetString("name")
	if e != nil {
		return "", "", e
	}
	if n == "" {
		return "", "", fmt.Errorf("--name is required")
	}
	return n, a, nil
}

func main() {

	root := &cobra.Command{Use: "az"}
	st := &cobra.Command{Use: "storage"}
	cc := &cobra.Command{Use: "container"}
	root.AddCommand(st)
	st.AddCommand(cc)
	cc.PersistentFlags().StringVar(&resourceGroup, "resource-group", "", "Resource group name")
	cc.PersistentFlags().StringVar(&endpoint, "mock-endpoint", "https://localhost:4567/", "Local mock endpoint")
	cc.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if resourceGroup == "" {
			return fmt.Errorf("--resource-group is required")
		}
		return nil
	}

	clients, err := storage.NewClients(subscriptionID, endpoint)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	svc := storage.NewContainerService(clients.BlobContainers)
	create := &cobra.Command{Use: "create", RunE: func(c *cobra.Command, a []string) error {
		n, x, e := values(c, true)
		if e != nil {
			return e
		}
		if e = svc.Create(context.Background(), resourceGroup, x, n); e != nil {
			return e
		}
		fmt.Println("Container created:", n)
		return nil
	}}

	addFlags(create, true)
	list := &cobra.Command{Use: "list", RunE: func(c *cobra.Command, a []string) error {

		_, accountName, e := values(c, false)

		if e != nil {
			fmt.Errorf("%s", e)
			return e
		}

		cs, e := svc.List(context.Background(), resourceGroup, accountName)
		if e != nil {
			fmt.Errorf("%s", e)
			return e
		}

		if len(cs) == 0 {
			fmt.Println("No containers found")
			return nil
		}

		fmt.Printf("Listing containers for account '%s':\n\n", accountName)
		fmt.Printf("%-20s %-15s %-20s\n", "NAME", "PUBLIC ACCESS", "DELETED?")
		fmt.Println("------------------------------------------------------------")

		for _, container := range cs {
			name := "Unknown"
			if container.Name != nil {
				name = *container.Name
			}

			publicAccess := "None"
			if container.Properties != nil && container.Properties.PublicAccess != nil {
				publicAccess = string(*container.Properties.PublicAccess)
			}

			isDeleted := false
			if container.Properties != nil && container.Properties.Deleted != nil {
				isDeleted = *container.Properties.Deleted
			}

			fmt.Printf("%-20s %-15s %-20t\n", name, publicAccess, isDeleted)
		}

		return nil
	}}

	addFlags(list, false)
	update := &cobra.Command{Use: "update", RunE: func(c *cobra.Command, a []string) error {
		n, x, e := values(c, true)
		if e != nil {
			return e
		}
		if e = svc.Update(context.Background(), resourceGroup, x, n); e != nil {
			return e
		}
		fmt.Println("Container updated:", n)
		return nil
	}}

	addFlags(update, true)
	show := &cobra.Command{Use: "show", RunE: func(c *cobra.Command, a []string) error {
		n, x, e := values(c, true)
		if e != nil {
			//fmt.Errorf("1- Error %s", e.Error())
			return e
		}
		v, e := svc.Get(context.Background(), resourceGroup, x, n)
		if e != nil {
			//fmt.Errorf("2- Error %s", e.Error())
			return e
		}

		//fmt.Printf("%V\n", v)
		fmt.Printf("Name: %s\n", *v.Name)
		return nil
	}}

	addFlags(show, true)
	exists := &cobra.Command{Use: "exists", RunE: func(c *cobra.Command, a []string) error {
		n, x, e := values(c, true)
		if e != nil {
			return e
		}
		v, e := svc.Exists(context.Background(), resourceGroup, x, n)
		if e != nil {
			return e
		}
		fmt.Println(v)
		return nil
	}}

	addFlags(exists, true)
	del := &cobra.Command{Use: "delete", RunE: func(c *cobra.Command, a []string) error {
		n, x, e := values(c, true)
		if e != nil {
			return e
		}
		if e = svc.Delete(context.Background(), resourceGroup, x, n); e != nil {
			return e
		}
		fmt.Println("Container deleted:", n)
		return nil
	}}

	addFlags(del, true)
	cc.AddCommand(create, list, update, show, exists, del)
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

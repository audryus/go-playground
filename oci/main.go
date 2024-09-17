package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/common"
	"github.com/oracle/oci-go-sdk/identity"
	"github.com/oracle/oci-go-sdk/objectstorage"
)

func main() {
	const tenancyOCID = "tenancyOCID"
	const userOCID = "userOCID"
	const region = "region"
	const fingerprint = "fingerprint"
	const compartmentOCID = "compartmentOCID"

	fileContent, err := os.ReadFile("ppk")
	fatalIfError(err)

	keyFile := string(fileContent)

	provider := common.NewRawConfigurationProvider(tenancyOCID, userOCID, region, fingerprint, keyFile, nil)

	c, err := identity.NewIdentityClientWithConfigurationProvider(provider)
	fatalIfError(err)

	request := identity.ListAvailabilityDomainsRequest{
		CompartmentId: common.String(string(tenancyOCID)),
	}

	ctx := context.Background()

	r, err := c.ListAvailabilityDomains(ctx, request)
	fatalIfError(err)
	fmt.Printf("List of available domains: %v\n", r.Items)

	osClient, err := objectstorage.NewObjectStorageClientWithConfigurationProvider(provider)
	fatalIfError(err)
	namespace := getNamespace(ctx, osClient)

	bucketName := "GoBucket"
	ensureBucketExists(ctx, osClient, namespace, bucketName, compartmentOCID)
	objectName := "lero/FreshObject.pdf"
	objectContent := "teste.pdf"
	err = putObject(ctx, osClient, namespace, bucketName, objectName, objectContent, nil)
	fatalIfError(err)
	name := "par-bucket-20240214-1711"
	auth := objectstorage.CreatePreauthenticatedRequestRequest{
		NamespaceName: &namespace,
		BucketName:    &bucketName,
		CreatePreauthenticatedRequestDetails: objectstorage.CreatePreauthenticatedRequestDetails{
			Name:       &name,
			AccessType: "AnyObjectRead",
			TimeExpires: &common.SDKTime{
				Time: time.Now().Add(time.Hour * 48),
			},
		},
	}
	res, err := osClient.CreatePreauthenticatedRequest(ctx, auth)
	fatalIfError(err)
	fmt.Printf("Authenticated %v", res)

	// defer deleteObject(ctx, osClient, namespace, bucketName, objectName)

	fmt.Println("go get object ", objectName)
	//contents, error := getObject(ctx, osClient, namespace, bucketName, objectName)
	//fatalIfError(error)
	//fmt.Println("Object contents: ", contents)

	// fmt.Println("Go Sleep")
	// time.Sleep(2 * time.Minute)
}

func deleteObject(ctx context.Context, c objectstorage.ObjectStorageClient, namespace, bucketname, objectname string) (err error) {
	request := objectstorage.DeleteObjectRequest{
		NamespaceName: &namespace,
		BucketName:    &bucketname,
		ObjectName:    &objectname,
	}
	_, err = c.DeleteObject(ctx, request)
	fatalIfError(err)
	fmt.Println("Deleted object ", objectname)
	return
}

func getObject(ctx context.Context, c objectstorage.ObjectStorageClient, namespace string, bucketname string, objectname string) (string, error) {
	fmt.Println("get object ", objectname)
	request := objectstorage.GetObjectRequest{
		NamespaceName: &namespace,
		BucketName:    &bucketname,
		ObjectName:    &objectname,
	}
	response, err := c.GetObject(ctx, request)
	fatalIfError(err)

	buf := new(strings.Builder)
	_, err = io.Copy(buf, response.Content)
	return buf.String(), err
}

func putObject(ctx context.Context, c objectstorage.ObjectStorageClient, namespace, bucketname, objectname, object string, metadata map[string]string) error {
	f, err := os.Open(object)
	fatalIfError(err)
	fi, err := f.Stat()
	fatalIfError(err)
	size := fi.Size()

	request := objectstorage.PutObjectRequest{
		NamespaceName: &namespace,
		BucketName:    &bucketname,
		ObjectName:    &objectname,
		ContentLength: &size,
		PutObjectBody: io.NopCloser(bufio.NewReader(f)),
		OpcMeta:       metadata,
	}
	_, err = c.PutObject(ctx, request)
	fatalIfError(err)

	fmt.Println("Put object ", objectname, " in bucket ", bucketname)
	return err
}

func ensureBucketExists(ctx context.Context, client objectstorage.ObjectStorageClient, namespace, name string, compartmentOCID string) {
	req := objectstorage.GetBucketRequest{
		NamespaceName: &namespace,
		BucketName:    &name,
	}
	// verify if bucket exists
	response, err := client.GetBucket(context.Background(), req)
	if err != nil {
		if 404 == response.RawResponse.StatusCode {
			createBucket(ctx, client, namespace, name, compartmentOCID)
			return
		}
		fatalIfError(err)
	}
}

func createBucket(ctx context.Context, client objectstorage.ObjectStorageClient, namespace string, name string, compartmentOCID string) {
	request := objectstorage.CreateBucketRequest{
		NamespaceName: &namespace,
	}
	request.CompartmentId = &compartmentOCID
	request.Name = &name
	request.Metadata = make(map[string]string)
	request.PublicAccessType = objectstorage.CreateBucketDetailsPublicAccessTypeNopublicaccess
	_, err := client.CreateBucket(ctx, request)
	fatalIfError(err)
	fmt.Println("Created bucket ", name)
}

func getNamespace(ctx context.Context, client objectstorage.ObjectStorageClient) string {
	request := objectstorage.GetNamespaceRequest{}
	r, err := client.GetNamespace(ctx, request)
	fatalIfError(err)
	return *r.Value
}

func fatalIfError(err error) {
	if err != nil {
		log.Fatalln(err.Error())
	}
}

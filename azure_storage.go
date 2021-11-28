package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image"
	"image/jpeg"
	"net/url"

	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/segmentio/ksuid"
)

const containerName = ""
const blobName = ""

func accInfo() (string, string, string, string) {
	azrKey := "azureKEY"
	azrBlobAccountName := "BlobAccName"
	azrPrimaryBlobServiceEndpoint := fmt.Sprintf("link/", azrBlobAccountName)
	azrBlobContainer := "container"

	return azrKey, azrBlobAccountName, azrPrimaryBlobServiceEndpoint, azrBlobContainer
}

//GetBlobName - dd
func GetBlobName() string {
	uuid := ksuid.New().String()
	return "i" + uuid
	//return fmt.Sprintf("%s-%v.jpg", t.Format("20060102"), uuid)
}

//UploadBytesToBlob - dd
func UploadBytesToBlob(b []byte) (string, error) {
	azrKey, accountName, endPoint, container := accInfo()                  // This is our account info method
	u, _ := url.Parse(fmt.Sprint(endPoint, container, "/", GetBlobName())) // This uses our Blob Name Generator to create individual blob urls
	credential, errC := azblob.NewSharedKeyCredential(accountName, azrKey) // Finally we create the credentials object required by the uploader
	if errC != nil {
		return "", errC
	}

	// Another Azure Specific object, which combines our generated URL and credentials
	blockBlobURL := azblob.NewBlockBlobURL(*u, azblob.NewPipeline(credential, azblob.PipelineOptions{}))

	ctx := context.Background() // We create an empty context (https://golang.org/pkg/context/#Background)

	// Provide any needed options to UploadToBlockBlobOptions (https://godoc.org/github.com/Azure/azure-storage-blob-go/azblob#UploadToBlockBlobOptions)
	o := azblob.UploadToBlockBlobOptions{
		BlobHTTPHeaders: azblob.BlobHTTPHeaders{
			ContentType: "image/jpg", //  Add any needed headers here
		},
	}

	// Combine all the pieces and perform the upload using UploadBufferToBlockBlob (https://godoc.org/github.com/Azure/azure-storage-blob-go/azblob#UploadBufferToBlockBlob)
	_, errU := azblob.UploadBufferToBlockBlob(ctx, b, blockBlobURL, o)
	return blockBlobURL.String(), errU
}

func b64ToJpeg(b64Image string) image.Image {
	unbased, _ := base64.StdEncoding.DecodeString(b64Image)
	res := bytes.NewReader(unbased)
	image, err := jpeg.Decode(res)
	if err != nil {
		fmt.Println("error converting to jpeg")
	}
	return image
}

//JpegToBytes - dd
func JpegToBytes(nImage image.Image) []byte {
	buf := new(bytes.Buffer)
	err := jpeg.Encode(buf, nImage, nil)
	if err != nil {
		fmt.Println("error transforming image")
	}
	return buf.Bytes()
}

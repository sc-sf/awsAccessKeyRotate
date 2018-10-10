package main

import (
	"fmt"
	"io/ioutil"
	"github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/aws/credentials"    
    "github.com/aws/aws-sdk-go/aws/session"
    "github.com/aws/aws-sdk-go/service/iam"
) 

func writeCredentialsFile(newKey, newSecret, cpath string) (error){
	credkey := []byte("[default]\n" +
		"aws_access_key_id = \"" + newKey + "\"\n" + 
		"aws_secret_access_key = \"" + newSecret + "\"\n")
	err := ioutil.WriteFile(cpath + "credentials", credkey, 0600)
	return err
}
func createNewKey(iamclient *iam.IAM, user string) {
	keyCreated, err := iamclient.CreateAccessKey(&iam.CreateAccessKeyInput{
		UserName: aws.String(user),
	})
	if err != nil { fmt.Println("keyCreated Err", err) }

	//fmt.Println("Success key creation", *keyCreated.AccessKey)
	fmt.Println("Success key creation", *keyCreated)
}

func getIamClient(keyid, keysecret, regn string) *iam.IAM {
	
		sess, err := session.NewSession(&aws.Config{
			Region:      aws.String(regn),
			Credentials: credentials.NewStaticCredentials(
				keyid, 
				keysecret, 
				""),
		})
		if err != nil { fmt.Println("sess Err", err) }
	
		// Create a IAM service client.
		return iam.New(sess)
}
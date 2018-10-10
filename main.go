package main

import (
    "fmt"
    "os"
    "flag"
    "time"

    "github.com/pelletier/go-toml"
    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/service/iam"
)
// GOOS=darwin go build -ldflags="-s -w" github.com/sc-sf/rotatekeys
// ./rotatekeys -username schoy -region us-west-2

func main() {

    iamUser := flag.String("username", "nobody", "Specify your IAM user name here")
    myregion := flag.String("region", "eu-west-1", "Specify the region here")

    flag.Parse()

    confPath := os.Getenv("HOME") + "/.aws/"
    cred, err := toml.LoadFile( confPath + "credentials")
    if err != nil { fmt.Println("Error: ", err); return}    
    key_id := cred.Get("default.aws_access_key_id").(string)
    key_secret := cred.Get("default.aws_secret_access_key").(string)
 
    iamClient := getIamClient(
        key_id, key_secret, *myregion,
    )

    listKeys, err := iamClient.ListAccessKeys(&iam.ListAccessKeysInput{
        MaxItems: aws.Int64(2),
        UserName: aws.String(*iamUser),
    })
    if err != nil { fmt.Println("Error from listKeys:\n", err) ; return}
    fmt.Println(*listKeys)

    // Each IAM user can have a maximum of 2 active access keys coexistent at any time.
    // There are just two situations: user has only one key or user has two keys

    // If there is only one access key and it's at least 6 hours old, 
    // create a new key for rotation.
    if len(listKeys.AccessKeyMetadata) < 2 { 
        
        // Don't rotate if this key is less than 6 hours old 
        for _, acekey := range listKeys.AccessKeyMetadata {
            now := time.Now()
            diff := now.Sub(*acekey.CreateDate)
            fmt.Println(diff.Hours())
            if diff.Hours() < 6 {
                return 
            }
        }
        keyCreated, err := iamClient.CreateAccessKey(&iam.CreateAccessKeyInput{
            UserName: aws.String(*iamUser),
        })
        if err != nil { fmt.Println("Error from keyCreated:\n", err) ; return}


        if err := writeCredentialsFile(*keyCreated.AccessKey.AccessKeyId, 
            *keyCreated.AccessKey.SecretAccessKey,
            confPath);
            err != nil { fmt.Println("Error from writeCredentialsFile:\n", err) ; return
        }

        _, err = iamClient.DeleteAccessKey(&iam.DeleteAccessKeyInput{
            AccessKeyId: aws.String(key_id),
            UserName:    aws.String(*iamUser),
        })
        if err != nil { fmt.Println("Error from keyDeleted", err); return }

    } else { 
        // When there are now two keys, 
        // delete one key first and then create a new one.
        var accessKeys []string
        listAccessKeysParams := &iam.ListAccessKeysInput{UserName: aws.String(*iamUser),}

        listAccessKeysFunc := func(page *iam.ListAccessKeysOutput, lastPage bool) (shouldContinue bool) {
			for _, k := range page.AccessKeyMetadata {
				accessKeys = append(accessKeys, *k.AccessKeyId)
			}
			return !lastPage
		}
        err = iamClient.ListAccessKeysPages(listAccessKeysParams, listAccessKeysFunc)
		if err != nil {fmt.Println("Error from ListAccessKeysPages", err); return}
        
        for _, k := range accessKeys {
        // We first only deletes the key that is not the "key_id" key used above 
        // to get the iam client session:
            if key_id != k {  
			    _, err = iamClient.DeleteAccessKey(&iam.DeleteAccessKeyInput{
				    UserName:    aws.String(*iamUser),
				    AccessKeyId: aws.String(k),
			    })
			    if err != nil {fmt.Println("Error from DeleteAccessKey", err); return}
            }
        }
        // Create new key
        keyCreated, err := iamClient.CreateAccessKey(&iam.CreateAccessKeyInput{
            UserName: aws.String(*iamUser),
        })
        if err != nil { fmt.Println("Error from keyCreated:\n", err) ; return}

        // Write to credentials file
        if err := writeCredentialsFile(*keyCreated.AccessKey.AccessKeyId, 
            *keyCreated.AccessKey.SecretAccessKey,
            confPath);
            err != nil { fmt.Println("Error from writeCredentialsFile:\n", err) ; return
        }
        // Now we can delete the "key_id" key now
        _, err = iamClient.DeleteAccessKey(&iam.DeleteAccessKeyInput{
            UserName:    aws.String(*iamUser),
            AccessKeyId: aws.String(key_id),
        })
        if err != nil {fmt.Println("Error from DeleteAccessKey", err); return}
    }

}


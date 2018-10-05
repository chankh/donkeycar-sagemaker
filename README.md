# Donkeycar SageMaker Workshop

This is a self-paced workshop designed for anyone who is interested in building self-driving cars
using Amazon SageMaker.

AWS has published several blog posts that walkthrough the process of building one. In the [first blog
post](https://aws.amazon.com/blogs/ai/build-an-autonomous-vehicle-on-aws-and-race-it-at-the-reinvent-robocar-rally/) 
of the autonomous vehicle series, you built your Donkey vehicle and deployed your pilot server onto
an Amazon EC2 instance. In the [second blog post](https://aws.amazon.com/blogs/ai/build-an-autonomous-vehicle-part-2-driving-your-vehicle/), you
learned to drive the Donkey car, and the Donkey car learned to self-drive. In the [third blog post](https://aws.amazon.com/blogs/ai/building-an-autonomous-vehicle-part-3-connecting-your-autonomous-vehicle/),
you learned about the process of streaming telemetry from the Donkey vehicle into AWS using AWS IoT.
In the [forth blog post](https://aws.amazon.com/blogs/machine-learning/building-an-autonomous-vehicle-part-4-using-behavioral-cloning-with-apache-mxnet-for-your-self-driving-car/),
you learned the concept of behavioral cloning with Convolutional Neural Networks (CNNs).

In this workshop, we will go through how to setup Amazon SageMaker with custom algorithm to train
the model for your autonomous vehicle.

## Prerequisites

You have built a car by following the instructions in [Assemble
hardware](http://docs.donkeycar.com/guide/build_hardware/) and [Install
software](http://docs.donkeycar.com/guide/install_software/) section of the [documentation from
donkeycar](http://docs.donkeycar.com/), with one exception that you use the repo from
https://github.com/chankh/donkey instead.

```
$ git clone https://github.com/chankh/donkey donkeycar 
$ pip install -e donkeycar
```

You will also need to have an AWS account to go through this workshop. 

## Setting up your AWS environment

Previously we would copy the data from the Pi to our Amazon EC2 instance. However with Amazon
SageMaker, you don't need an Amazon EC2 instance. Instead the data will be pulled from Amazon S3. So
let's first create our stack containing all required AWS resources by choosing one of the regions
below.

| Region | Launch Template |
| ------------- | ------------- |
| **N. Virginia** (us-east-1) | [<img src="images/deploy-to-aws.png">](https://console.aws.amazon.com/cloudformation/home?region=us-east-1/stacks/new?stackName=donkeycar&templateURL=https://s3.amazonaws.com/khk-us-east-1/sagemaker/donkeycar-workshop.yaml) |
| **Oregon** (us-west-2) | [<img src="images/deploy-to-aws.png">](https://console.aws.amazon.com/cloudformation/home?region=us-west-2/stacks/new?stackName=donkeycar&templateURL=https://s3.amazonaws.com/khk-us-east-1/sagemaker/donkeycar-workshop.yaml) |
| **Tokyo** (ap-northeast-1) | [<img src="images/deploy-to-aws.png">](https://console.aws.amazon.com/cloudformation/home?region=ap-northeast-1/stacks/new?stackName=donkeycar&templateURL=https://s3.amazonaws.com/khk-us-east-1/sagemaker/donkeycar-workshop.yaml) |

Once the stack is created, status is **CREATE_COMPLETE**, you can find the _AWS IoT Endpoint URL_
under the **Outputs** tab.



## Configuring your Donkeycar

You will need to calibrate your car by following the instructions described in [Calibrate your
car](http://docs.donkeycar.com/guide/calibrate/).

### Setting up Donkeycar with AWS IoT

Additional configuration for integration with AWS would 

```
#AWS IOT
IOT_ENABLED = True
VEHICLE_ID = 'donkey'
AWS_ENDPOINT = [replace with endpoint from AWS IoT console]
CA_PATH = /home/pi/aws/ca.cert
PRIVATE_KEY_PATH = /home/pi/aws/key.pem
CERTIFICATE_PATH = /home/pi/aws/cert.pem
```


## Collecting Data



## Training the model with Amazon SageMaker

Training the model with Amazon SageMaker

The *SageMaker Python SDK* makes it easy to train and deploy ML models. In this notebook we train a
model from data collected from the robocar.

### Setup the environment

First we need to define a few variables that will be needed later. Here we specify a bucket to use
and the role that will be used for working with SageMaker.

```python
import sagemaker as sage
from time import gmtime, strftime

#Bucket with input data
data_location = 's3://autonomous-vehicles'

#IAM execution role that gives SageMaker access to resources in your AWS account.
#We can use the SageMaker Python SDK to get the role from our notebook environment. 
role = sage.get_execution_role()

#The session remembers our connection parameters to SageMaker. We'll use it to 
#perform all of our SageMaker operations.
sess = sage.Session()
```

### Create an estimator and fit the model

In order to use SageMaker to fit our algorithm, we'll create an Estimator that defines how to use
the container to train. This includes the configuration we need to invoke SageMaker training:

* The container name. This is constructed as in the shell commands above.
* The role. As defined above.
* The instance count which is the number of machines to use for training.
* The instance type which is the type of machine to use for training.
* The output path determines where the model artifact will be written.
* The session is the SageMaker session object that we defined above. Then we use fit() on the
  estimator to train against the data that we uploaded above.

```python
account = sess.boto_session.client('sts').get_caller_identity()['Account']
region = sess.boto_session.region_name
image = '{}.dkr.ecr.{}.amazonaws.com/donkey:latest'.format(account, region)

tree = sage.estimator.Estimator(image,
                       role, 1, 'ml.c5.2xlarge',
                       output_path="s3://{}/output".format(sess.default_bucket()),
                       sagemaker_session=sess)

tree.fit(data_location)
```

Download the model and deploy to robocar

Amazon SageMaker saves the model to a S3 location. You can now download the model from S3 into the Pi
and extract the contents from the tarball. This should give you a file named donkeycar.

```
$ curl -O [S3 location]
$ tar -zxf model.tar.gz
```

Run your autonomous vehicle using the downloaded model.

```
$ python manage.py drive --model donkeycar
```

Conclusion

Amazon SageMaker provides an easy to use platform for building ML models. In this post, you've seen
how it's possible to bring your own algorithms to SageMaker and start training immediately. It is so
easy that even someone without knowledge on ML is able to do it without any special training.


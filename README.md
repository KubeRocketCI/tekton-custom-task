# Tekton Custom Task

## Introduction

Tekton Custom Task extends Kubernetes and Tekton capabilities by providing custom task implementations.
These custom tasks are designed to facilitate complex workflows that standard Tekton tasks may not cover,
making it a powerful tool for DevOps teams looking to extend their CI/CD pipelines with custom logic.

## Project Structure

- **api/v1alpha1/**: Contains the API definitions for the custom tasks, including the structure and validation of custom resource definitions (CRDs).
- **cmd/**: Hosts the main application entry point and the command-line interface setup.
- **config/**: Includes Kubernetes configuration files for deploying the custom tasks, such as CRDs, RBAC rules, and sample configurations.
- **docs/**: Provides detailed documentation on the API and usage examples.
- **deploy-templates/**: Contains Helm chart templates for deploying the custom tasks controller.

## Getting Started

To get started with Tekton Custom Task, ensure you have Kubernetes and Tekton Pipelines installed in your environment. Follow these steps to deploy a custom task:

1. Clone the repository to your local environment.
2. Navigate to the `config/` directory.
3. Apply the CRDs to your Kubernetes cluster:

   ```bash
   kubectl apply -f config/crd/bases/
   ```

4. Deploy the custom tasks using:

   ```bash
   kubectl apply -f config/samples/
   ```

Below is an example of how to define a custom task in your Tekton pipeline:

```yaml
apiVersion: edp.epam.com/v1alpha1
kind: ApprovalTask
metadata:
  name: approvaltask-sample
  labels:
    app.kubernetes.io/name: tekton-custom-task
    app.kubernetes.io/managed-by: kustomize
spec:
  action: Pending
  description: "Approval required for the next step in the pipeline."
```

This example demonstrates how to run a `CustomTask` named `example-custom-task` within a Tekton pipeline.

## Usage

This set of instructions guides you through the process of creating a custom `ApprovalTask` and incorporating it into a Tekton pipeline to serve as an approval step.

To integrate the `ApprovalTask` into your CI/CD pipelines, follow these steps to define the custom task and use it within a Tekton pipeline.

### Defining an `ApprovalTask`

Create an `ApprovalTask` definition by preparing a YAML file named `approvaltask.yaml` with the contents below:

```yaml
apiVersion: edp.epam.com/v1alpha1
kind: ApprovalTask
metadata:
  name: approvaltask-example
spec:
  action: Pending
  description: "Approval required for the next deployment phase."
```

This YAML snippet creates an `ApprovalTask` resource named `approvaltask-example`. The task is initially set to a `Pending` state, indicating it awaits approval, and it includes a description to provide context for the approval required.

### Using the `ApprovalTask` in a Tekton Pipeline

Incorporate the `ApprovalTask` within a Tekton pipeline by defining both a `Pipeline` and a `PipelineRun`.

#### Define the Pipeline

Create a file named `pipeline.yaml` with the following content:

```yaml
apiVersion: tekton.dev/v1beta1
kind: Pipeline
metadata:
  name: example-pipeline
spec:
  tasks:
    - name: approval-task
      taskRef:
        apiVersion: edp.epam.com/v1alpha1
        kind: ApprovalTask
        name: approvaltask-example
```

This pipeline, named `example-pipeline`, contains a single task named `approval-task`. This task references your previously defined `ApprovalTask` (`approvaltask-example`).

#### Define the PipelineRun

To initiate the pipeline execution, define a `PipelineRun` in a file named `pipeline-run.yaml` with the contents below:

```yaml
apiVersion: tekton.dev/v1beta1
kind: PipelineRun
metadata:
  name: example-pipeline-run
spec:
  pipelineRef:
    name: example-pipeline
```

This `PipelineRun`, named `example-pipeline-run`, triggers the execution of `example-pipeline`.

### Applying the Definitions

Deploy the `ApprovalTask`, the pipeline, and initiate the pipeline run by applying the YAML files to your Kubernetes cluster:

```sh
kubectl apply -f approvaltask.yaml
kubectl apply -f pipeline.yaml
kubectl apply -f pipeline-run.yaml
```

## Features

- **Custom Task Definitions**: Define your tasks that extend the Tekton pipeline model.
- **Flexible Configuration**: Leverage Kubernetes CRDs for task definitions, allowing for dynamic and flexible configurations.
- **Integration with Tekton Pipelines**: Seamlessly integrate custom tasks within your existing Tekton pipelines.

## Integration and Notifications

Tekton Custom Task supports integration with Kubernetes dashboards and notification systems. This allows for improved monitoring and alerting capabilities for your custom tasks.

## Roadmap

Future developments include:

- Enhanced UI for managing custom tasks.
- More examples and templates for common use cases.
- Improved integration with external tools and platforms.

## Contributing

We welcome contributions in the form of issues and pull requests. Please follow the contributing guidelines outlined in the repository.

## License

This project is licensed under the [Apache License 2.0](LICENSE.txt).

package aws

import (
	"fmt"
	"log"
	"regexp"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/private/protocol"
	"github.com/aws/aws-sdk-go/service/sagemaker"
	"github.com/hashicorp/aws-sdk-go-base/tfawserr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/structure"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/terraform-providers/terraform-provider-aws/aws/internal/keyvaluetags"
	iamwaiter "github.com/terraform-providers/terraform-provider-aws/aws/internal/service/iam/waiter"
	"github.com/terraform-providers/terraform-provider-aws/aws/internal/service/sagemaker/finder"
	"github.com/terraform-providers/terraform-provider-aws/aws/internal/service/sagemaker/waiter"
	"github.com/terraform-providers/terraform-provider-aws/aws/internal/tfresource"
)

func resourceAwsSagemakerFlowDefinition() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsSagemakerFlowDefinitionCreate,
		Read:   resourceAwsSagemakerFlowDefinitionRead,
		Update: resourceAwsSagemakerFlowDefinitionUpdate,
		Delete: resourceAwsSagemakerFlowDefinitionDelete,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"flow_definition_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateFunc: validation.All(
					validation.StringLenBetween(1, 63),
					validation.StringMatch(regexp.MustCompile(`^[a-z0-9](-*[a-z0-9])*$`), "Valid characters are a-z, 0-9, and - (hyphen)."),
				),
			},
			"human_loop_activation_config": {
				Type:         schema.TypeList,
				Optional:     true,
				ForceNew:     true,
				MaxItems:     1,
				RequiredWith: []string{"human_loop_request_source", "human_loop_activation_config"},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"human_loop_activation_conditions_config": {
							Type:     schema.TypeList,
							Optional: true,
							ForceNew: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"human_loop_activation_conditions": {
										Type:     schema.TypeString,
										Required: true,
										ForceNew: true,
										ValidateFunc: validation.All(
											validation.StringLenBetween(1, 10240),
											validation.StringIsJSON,
										),
										StateFunc: func(v interface{}) string {
											json, _ := structure.NormalizeJsonString(v)
											return json
										},
										DiffSuppressFunc: suppressEquivalentJsonDiffs,
									},
								},
							},
						},
					},
				},
			},
			"human_loop_config": {
				Type:     schema.TypeList,
				Required: true,
				ForceNew: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"human_task_ui_arn": {
							Type:         schema.TypeString,
							Required:     true,
							ForceNew:     true,
							ValidateFunc: validateArn,
						},
						"public_workforce_task_price": {
							Type:     schema.TypeList,
							Optional: true,
							ForceNew: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"amount_in_usd": {
										Type:     schema.TypeList,
										Optional: true,
										ForceNew: true,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"cents": {
													Type:         schema.TypeInt,
													Optional:     true,
													ForceNew:     true,
													ValidateFunc: validation.IntBetween(0, 99),
												},
												"dollars": {
													Type:         schema.TypeInt,
													Optional:     true,
													ForceNew:     true,
													ValidateFunc: validation.IntBetween(0, 2),
												},
												"tenth_fractions_of_a_cent": {
													Type:         schema.TypeInt,
													Optional:     true,
													ForceNew:     true,
													ValidateFunc: validation.IntBetween(0, 9),
												},
											},
										},
									},
								},
							},
						},
						"task_availability_lifetime_in_seconds": {
							Type:         schema.TypeInt,
							Optional:     true,
							ForceNew:     true,
							ValidateFunc: validation.IntBetween(1, 864000),
						},
						"task_count": {
							Type:         schema.TypeInt,
							Required:     true,
							ForceNew:     true,
							ValidateFunc: validation.IntBetween(1, 3),
						},
						"task_description": {
							Type:         schema.TypeString,
							Required:     true,
							ForceNew:     true,
							ValidateFunc: validation.StringLenBetween(1, 255),
						},
						"task_keywords": {
							Type:     schema.TypeSet,
							Optional: true,
							MinItems: 1,
							MaxItems: 5,
							Elem: &schema.Schema{
								Type: schema.TypeString,
								ValidateFunc: validation.All(
									validation.StringLenBetween(1, 30),
									validation.StringMatch(regexp.MustCompile(`^[A-Za-z0-9]+( [A-Za-z0-9]+)*$`), ""),
								),
							},
						},
						"task_time_limit_in_seconds": {
							Type:         schema.TypeInt,
							Optional:     true,
							ForceNew:     true,
							Default:      3600,
							ValidateFunc: validation.IntBetween(30, 28800),
						},
						"task_title": {
							Type:         schema.TypeString,
							Required:     true,
							ForceNew:     true,
							ValidateFunc: validation.StringLenBetween(1, 128),
						},
						"workteam_arn": {
							Type:         schema.TypeString,
							Required:     true,
							ForceNew:     true,
							ValidateFunc: validateArn,
						},
					},
				},
			},
			"human_loop_request_source": {
				Type:         schema.TypeList,
				Optional:     true,
				ForceNew:     true,
				MaxItems:     1,
				RequiredWith: []string{"human_loop_request_source", "human_loop_activation_config"},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"aws_managed_human_loop_request_source": {
							Type:         schema.TypeString,
							Required:     true,
							ForceNew:     true,
							ValidateFunc: validation.StringInSlice(sagemaker.AwsManagedHumanLoopRequestSource_Values(), false),
						},
					},
				},
			},
			"output_config": {
				Type:     schema.TypeList,
				Required: true,
				ForceNew: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"kms_key_id": {
							Type:         schema.TypeString,
							Optional:     true,
							ForceNew:     true,
							ValidateFunc: validateArn,
						},
						"s3_output_path": {
							Type:     schema.TypeString,
							ForceNew: true,
							Required: true,
							ValidateFunc: validation.All(
								validation.StringMatch(regexp.MustCompile(`^(https|s3)://([^/])/?(.*)$`), ""),
								validation.StringLenBetween(1, 512),
							),
						},
					},
				},
			},
			"role_arn": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateArn,
			},
			"tags":     tagsSchema(),
			"tags_all": tagsSchemaComputed(),
		},

		CustomizeDiff: SetTagsDiff,
	}
}

func resourceAwsSagemakerFlowDefinitionCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).sagemakerconn
	defaultTagsConfig := meta.(*AWSClient).DefaultTagsConfig
	tags := defaultTagsConfig.MergeTags(keyvaluetags.New(d.Get("tags").(map[string]interface{})))

	name := d.Get("flow_definition_name").(string)
	input := &sagemaker.CreateFlowDefinitionInput{
		FlowDefinitionName: aws.String(name),
		HumanLoopConfig:    expandSagemakerFlowDefinitionHumanLoopConfig(d.Get("human_loop_config").([]interface{})),
		RoleArn:            aws.String(d.Get("role_arn").(string)),
		OutputConfig:       expandSagemakerFlowDefinitionOutputConfig(d.Get("output_config").([]interface{})),
	}

	if v, ok := d.GetOk("human_loop_activation_config"); ok && (len(v.([]interface{})) > 0) {
		loopConfig, err := expandSagemakerFlowDefinitionHumanLoopActivationConfig(v.([]interface{}))
		if err != nil {
			return fmt.Errorf("error creating SageMaker Flow Definition Human Loop Activation Config (%s): %w", name, err)
		}
		input.HumanLoopActivationConfig = loopConfig
	}

	if v, ok := d.GetOk("human_loop_request_source"); ok && (len(v.([]interface{})) > 0) {
		input.HumanLoopRequestSource = expandSagemakerFlowDefinitionHumanLoopRequestSource(v.([]interface{}))
	}

	if len(tags) > 0 {
		input.Tags = tags.IgnoreAws().SagemakerTags()
	}

	log.Printf("[DEBUG] Creating SageMaker Flow Definition: %s", input)
	_, err := tfresource.RetryWhenAwsErrCodeEquals(iamwaiter.PropagationTimeout, func() (interface{}, error) {
		return conn.CreateFlowDefinition(input)
	}, "ValidationException")

	if err != nil {
		return fmt.Errorf("error creating SageMaker Flow Definition (%s): %w", name, err)
	}

	d.SetId(name)

	if _, err := waiter.FlowDefinitionActive(conn, d.Id()); err != nil {
		return fmt.Errorf("error waiting for SageMaker Flow Definition (%s) to become active: %w", d.Id(), err)
	}

	return resourceAwsSagemakerFlowDefinitionRead(d, meta)
}

func resourceAwsSagemakerFlowDefinitionRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).sagemakerconn
	defaultTagsConfig := meta.(*AWSClient).DefaultTagsConfig
	ignoreTagsConfig := meta.(*AWSClient).IgnoreTagsConfig

	flowDefinition, err := finder.FlowDefinitionByName(conn, d.Id())

	if !d.IsNewResource() && tfresource.NotFound(err) {
		log.Printf("[WARN] SageMaker Flow Definition (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err != nil {
		return fmt.Errorf("error reading SageMaker Flow Definition (%s): %w", d.Id(), err)
	}

	arn := aws.StringValue(flowDefinition.FlowDefinitionArn)
	d.Set("arn", arn)
	d.Set("role_arn", flowDefinition.RoleArn)
	d.Set("flow_definition_name", flowDefinition.FlowDefinitionName)

	if err := d.Set("human_loop_activation_config", flattenSagemakerFlowDefinitionHumanLoopActivationConfig(flowDefinition.HumanLoopActivationConfig)); err != nil {
		return fmt.Errorf("error setting human_loop_activation_config: %w", err)
	}

	if err := d.Set("human_loop_config", flattenSagemakerFlowDefinitionHumanLoopConfig(flowDefinition.HumanLoopConfig)); err != nil {
		return fmt.Errorf("error setting human_loop_config: %w", err)
	}

	if err := d.Set("human_loop_request_source", flattenSagemakerFlowDefinitionHumanLoopRequestSource(flowDefinition.HumanLoopRequestSource)); err != nil {
		return fmt.Errorf("error setting human_loop_request_source: %w", err)
	}

	if err := d.Set("output_config", flattenSagemakerFlowDefinitionOutputConfig(flowDefinition.OutputConfig)); err != nil {
		return fmt.Errorf("error setting output_config: %w", err)
	}

	tags, err := keyvaluetags.SagemakerListTags(conn, arn)

	if err != nil {
		return fmt.Errorf("error listing tags for SageMaker Flow Definition (%s): %w", d.Id(), err)
	}

	tags = tags.IgnoreAws().IgnoreConfig(ignoreTagsConfig)

	//lintignore:AWSR002
	if err := d.Set("tags", tags.RemoveDefaultConfig(defaultTagsConfig).Map()); err != nil {
		return fmt.Errorf("error setting tags: %w", err)
	}

	if err := d.Set("tags_all", tags.Map()); err != nil {
		return fmt.Errorf("error setting tags_all: %w", err)
	}

	return nil
}

func resourceAwsSagemakerFlowDefinitionUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).sagemakerconn

	if d.HasChange("tags_all") {
		o, n := d.GetChange("tags_all")

		if err := keyvaluetags.SagemakerUpdateTags(conn, d.Get("arn").(string), o, n); err != nil {
			return fmt.Errorf("error updating SageMaker Flow Definition (%s) tags: %w", d.Id(), err)
		}
	}

	return resourceAwsSagemakerFlowDefinitionRead(d, meta)
}

func resourceAwsSagemakerFlowDefinitionDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).sagemakerconn

	log.Printf("[DEBUG] Deleting SageMaker Flow Definition: %s", d.Id())
	_, err := conn.DeleteFlowDefinition(&sagemaker.DeleteFlowDefinitionInput{
		FlowDefinitionName: aws.String(d.Id()),
	})

	if tfawserr.ErrCodeEquals(err, sagemaker.ErrCodeResourceNotFound) {
		return nil
	}

	if err != nil {
		return fmt.Errorf("error deleting SageMaker Flow Definition (%s): %w", d.Id(), err)
	}

	if _, err := waiter.FlowDefinitionDeleted(conn, d.Id()); err != nil {
		return fmt.Errorf("error waiting for SageMaker Flow Definition (%s) to delete: %w", d.Id(), err)
	}

	return nil
}

func expandSagemakerFlowDefinitionHumanLoopActivationConfig(l []interface{}) (*sagemaker.HumanLoopActivationConfig, error) {
	if len(l) == 0 || l[0] == nil {
		return nil, nil
	}

	m := l[0].(map[string]interface{})

	loopConfig, err := expandSagemakerFlowDefinitionHumanLoopActivationConditionsConfig(m["human_loop_activation_conditions_config"].([]interface{}))
	if err != nil {
		return nil, err
	}
	config := &sagemaker.HumanLoopActivationConfig{
		HumanLoopActivationConditionsConfig: loopConfig,
	}

	return config, nil
}

func flattenSagemakerFlowDefinitionHumanLoopActivationConfig(config *sagemaker.HumanLoopActivationConfig) []map[string]interface{} {
	if config == nil {
		return []map[string]interface{}{}
	}

	m := map[string]interface{}{
		"human_loop_activation_conditions_config": flattenSagemakerFlowDefinitionHumanLoopActivationConditionsConfig(config.HumanLoopActivationConditionsConfig),
	}

	return []map[string]interface{}{m}
}

func expandSagemakerFlowDefinitionHumanLoopActivationConditionsConfig(l []interface{}) (*sagemaker.HumanLoopActivationConditionsConfig, error) {
	if len(l) == 0 || l[0] == nil {
		return nil, nil
	}

	m := l[0].(map[string]interface{})

	v, err := protocol.DecodeJSONValue(m["human_loop_activation_conditions"].(string), protocol.NoEscape)
	if err != nil {
		return nil, err
	}

	config := &sagemaker.HumanLoopActivationConditionsConfig{
		HumanLoopActivationConditions: v,
	}

	return config, nil
}

func flattenSagemakerFlowDefinitionHumanLoopActivationConditionsConfig(config *sagemaker.HumanLoopActivationConditionsConfig) []map[string]interface{} {
	if config == nil {
		return []map[string]interface{}{}
	}

	v, err := protocol.EncodeJSONValue(config.HumanLoopActivationConditions, protocol.NoEscape)
	if err != nil {
		return []map[string]interface{}{}
	}

	m := map[string]interface{}{
		"human_loop_activation_conditions": v,
	}

	return []map[string]interface{}{m}
}

func expandSagemakerFlowDefinitionOutputConfig(l []interface{}) *sagemaker.FlowDefinitionOutputConfig {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	m := l[0].(map[string]interface{})

	config := &sagemaker.FlowDefinitionOutputConfig{
		S3OutputPath: aws.String(m["s3_output_path"].(string)),
	}

	if v, ok := m["kms_key_id"].(string); ok && v != "" {
		config.KmsKeyId = aws.String(v)
	}

	return config
}

func flattenSagemakerFlowDefinitionOutputConfig(config *sagemaker.FlowDefinitionOutputConfig) []map[string]interface{} {
	if config == nil {
		return []map[string]interface{}{}
	}

	m := map[string]interface{}{
		"kms_key_id":     aws.StringValue(config.KmsKeyId),
		"s3_output_path": aws.StringValue(config.S3OutputPath),
	}

	return []map[string]interface{}{m}
}

func expandSagemakerFlowDefinitionHumanLoopRequestSource(l []interface{}) *sagemaker.HumanLoopRequestSource {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	m := l[0].(map[string]interface{})

	config := &sagemaker.HumanLoopRequestSource{
		AwsManagedHumanLoopRequestSource: aws.String(m["aws_managed_human_loop_request_source"].(string)),
	}

	return config
}

func flattenSagemakerFlowDefinitionHumanLoopRequestSource(config *sagemaker.HumanLoopRequestSource) []map[string]interface{} {
	if config == nil {
		return []map[string]interface{}{}
	}

	m := map[string]interface{}{
		"aws_managed_human_loop_request_source": aws.StringValue(config.AwsManagedHumanLoopRequestSource),
	}

	return []map[string]interface{}{m}
}

func expandSagemakerFlowDefinitionHumanLoopConfig(l []interface{}) *sagemaker.HumanLoopConfig {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	m := l[0].(map[string]interface{})

	config := &sagemaker.HumanLoopConfig{
		HumanTaskUiArn:  aws.String(m["human_task_ui_arn"].(string)),
		TaskCount:       aws.Int64(int64(m["task_count"].(int))),
		TaskDescription: aws.String(m["task_description"].(string)),
		TaskTitle:       aws.String(m["task_title"].(string)),
		WorkteamArn:     aws.String(m["workteam_arn"].(string)),
	}

	if v, ok := m["public_workforce_task_price"].([]interface{}); ok && len(v) > 0 {
		config.PublicWorkforceTaskPrice = expandSagemakerFlowDefinitionPublicWorkforceTaskPrice(v)
	}

	if v, ok := m["task_keywords"].(*schema.Set); ok && v.Len() > 0 {
		config.TaskKeywords = expandStringSet(v)
	}

	if v, ok := m["task_availability_lifetime_in_seconds"].(int); ok {
		config.TaskAvailabilityLifetimeInSeconds = aws.Int64(int64(v))
	}

	if v, ok := m["task_time_limit_in_seconds"].(int); ok {
		config.TaskTimeLimitInSeconds = aws.Int64(int64(v))
	}

	return config
}

func flattenSagemakerFlowDefinitionHumanLoopConfig(config *sagemaker.HumanLoopConfig) []map[string]interface{} {
	if config == nil {
		return []map[string]interface{}{}
	}

	m := map[string]interface{}{
		"human_task_ui_arn": aws.StringValue(config.HumanTaskUiArn),
		"task_count":        aws.Int64Value(config.TaskCount),
		"task_description":  aws.StringValue(config.TaskDescription),
		"task_title":        aws.StringValue(config.TaskTitle),
		"workteam_arn":      aws.StringValue(config.WorkteamArn),
	}

	if config.PublicWorkforceTaskPrice != nil {
		m["public_workforce_task_price"] = flattenSagemakerFlowDefinitionPublicWorkforceTaskPrice(config.PublicWorkforceTaskPrice)
	}

	if config.TaskKeywords != nil {
		m["task_keywords"] = flattenStringSet(config.TaskKeywords)
	}

	if config.TaskAvailabilityLifetimeInSeconds != nil {
		m["task_availability_lifetime_in_seconds"] = aws.Int64Value(config.TaskAvailabilityLifetimeInSeconds)
	}

	if config.TaskTimeLimitInSeconds != nil {
		m["task_time_limit_in_seconds"] = aws.Int64Value(config.TaskTimeLimitInSeconds)
	}

	return []map[string]interface{}{m}
}

func expandSagemakerFlowDefinitionPublicWorkforceTaskPrice(l []interface{}) *sagemaker.PublicWorkforceTaskPrice {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	m := l[0].(map[string]interface{})

	config := &sagemaker.PublicWorkforceTaskPrice{}

	if v, ok := m["amount_in_usd"].([]interface{}); ok && len(v) > 0 {
		config.AmountInUsd = expandSagemakerFlowDefinitionAmountInUsd(v)
	}

	return config
}

func expandSagemakerFlowDefinitionAmountInUsd(l []interface{}) *sagemaker.USD {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	m := l[0].(map[string]interface{})

	config := &sagemaker.USD{}

	if v, ok := m["cents"].(int); ok {
		config.Cents = aws.Int64(int64(v))
	}

	if v, ok := m["dollars"].(int); ok {
		config.Dollars = aws.Int64(int64(v))
	}

	if v, ok := m["tenth_fractions_of_a_cent"].(int); ok {
		config.TenthFractionsOfACent = aws.Int64(int64(v))
	}

	return config
}

func flattenSagemakerFlowDefinitionAmountInUsd(config *sagemaker.USD) []map[string]interface{} {
	if config == nil {
		return []map[string]interface{}{}
	}

	m := map[string]interface{}{}

	if config.Cents != nil {
		m["cents"] = aws.Int64Value(config.Cents)
	}

	if config.Dollars != nil {
		m["dollars"] = aws.Int64Value(config.Dollars)
	}

	if config.TenthFractionsOfACent != nil {
		m["tenth_fractions_of_a_cent"] = aws.Int64Value(config.TenthFractionsOfACent)
	}

	return []map[string]interface{}{m}
}

func flattenSagemakerFlowDefinitionPublicWorkforceTaskPrice(config *sagemaker.PublicWorkforceTaskPrice) []map[string]interface{} {
	if config == nil {
		return []map[string]interface{}{}
	}

	m := map[string]interface{}{}

	if config.AmountInUsd != nil {
		m["amount_in_usd"] = flattenSagemakerFlowDefinitionAmountInUsd(config.AmountInUsd)
	}

	return []map[string]interface{}{m}
}

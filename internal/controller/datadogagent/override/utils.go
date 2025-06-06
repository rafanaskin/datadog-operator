// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package override

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/DataDog/datadog-operator/api/datadoghq/v2alpha1"
	"github.com/DataDog/datadog-operator/pkg/constants"
)

func getDefaultConfigMapName(ddaName, fileName string) string {
	return fmt.Sprintf("%s-%s-yaml", ddaName, strings.Split(fileName, ".")[0])
}

func hasProbeHandler(probe *corev1.Probe) bool {
	handler := &probe.ProbeHandler
	if handler.Exec != nil || handler.HTTPGet != nil || handler.TCPSocket != nil || handler.GRPC != nil {
		return true
	}
	return false
}

func SetOverrideFromDDA(dda *v2alpha1.DatadogAgent, ddaiSpec *v2alpha1.DatadogAgentSpec) {
	if ddaiSpec == nil {
		ddaiSpec = &v2alpha1.DatadogAgentSpec{}
	}
	if ddaiSpec.Override == nil {
		ddaiSpec.Override = make(map[v2alpha1.ComponentName]*v2alpha1.DatadogAgentComponentOverride)
	}
	if _, ok := ddaiSpec.Override[v2alpha1.NodeAgentComponentName]; !ok {
		ddaiSpec.Override[v2alpha1.NodeAgentComponentName] = &v2alpha1.DatadogAgentComponentOverride{}
	}
	if ddaiSpec.Override[v2alpha1.NodeAgentComponentName].Labels == nil {
		ddaiSpec.Override[v2alpha1.NodeAgentComponentName].Labels = make(map[string]string)
	}
	// Set empty provider label
	ddaiSpec.Override[v2alpha1.NodeAgentComponentName].Labels[constants.MD5AgentDeploymentProviderLabelKey] = ""
}

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package sbom

import (
	"testing"

	apicommon "github.com/DataDog/datadog-operator/api/datadoghq/common"
	"github.com/DataDog/datadog-operator/api/datadoghq/v2alpha1"
	apiutils "github.com/DataDog/datadog-operator/api/utils"
	"github.com/DataDog/datadog-operator/internal/controller/datadogagent/common"
	"github.com/DataDog/datadog-operator/internal/controller/datadogagent/feature"
	"github.com/DataDog/datadog-operator/internal/controller/datadogagent/feature/fake"
	"github.com/DataDog/datadog-operator/internal/controller/datadogagent/feature/test"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func Test_sbomFeature_Configure(t *testing.T) {

	sbomDisabled := v2alpha1.DatadogAgent{
		Spec: v2alpha1.DatadogAgentSpec{
			Features: &v2alpha1.DatadogFeatures{
				SBOM: &v2alpha1.SBOMFeatureConfig{
					Enabled: apiutils.NewBoolPointer(false),
				},
			},
		},
	}
	sbomEnabled := sbomDisabled.DeepCopy()
	{
		sbomEnabled.Spec.Features.SBOM.Enabled = apiutils.NewBoolPointer(true)
	}

	sbomEnabledContainerImageEnabled := sbomEnabled.DeepCopy()
	{
		sbomEnabledContainerImageEnabled.Spec.Features.SBOM.ContainerImage = &v2alpha1.SBOMContainerImageConfig{Enabled: apiutils.NewBoolPointer(true)}
	}

	sbomEnabledContainerImageOverlayFSEnabled := sbomEnabled.DeepCopy()
	{
		sbomEnabledContainerImageOverlayFSEnabled.Spec.Features.SBOM.ContainerImage = &v2alpha1.SBOMContainerImageConfig{Enabled: apiutils.NewBoolPointer(true), UncompressedLayersSupport: true, OverlayFSDirectScan: true}
	}

	sbomEnabledHostEnabled := sbomEnabled.DeepCopy()
	{
		sbomEnabledHostEnabled.Spec.Features.SBOM.Host = &v2alpha1.SBOMHostConfig{Enabled: apiutils.NewBoolPointer(true)}
	}

	sbomNodeAgentWantFunc := func(t testing.TB, mgrInterface feature.PodTemplateManagers) {
		mgr := mgrInterface.(*fake.PodTemplateManagers)

		wantEnvVars := []*corev1.EnvVar{
			{
				Name:  DDSBOMEnabled,
				Value: "true",
			},
			{
				Name:  DDSBOMContainerImageEnabled,
				Value: "false",
			},
			{
				Name:  DDSBOMHostEnabled,
				Value: "false",
			},
		}

		nodeCoreAgentEnvVars := mgr.EnvVarMgr.EnvVarsByC[apicommon.CoreAgentContainerName]
		assert.True(t, apiutils.IsEqualStruct(nodeCoreAgentEnvVars, wantEnvVars), "Core agent envvars \ndiff = %s", cmp.Diff(nodeCoreAgentEnvVars, wantEnvVars))
	}

	sbomWithContainerImageWantFunc := func(t testing.TB, mgrInterface feature.PodTemplateManagers) {
		mgr := mgrInterface.(*fake.PodTemplateManagers)

		wantEnvVars := []*corev1.EnvVar{
			{
				Name:  DDSBOMEnabled,
				Value: "true",
			},
			{
				Name:  DDSBOMContainerImageEnabled,
				Value: "true",
			},
			{
				Name:  DDSBOMHostEnabled,
				Value: "false",
			},
		}

		nodeCoreAgentEnvVars := mgr.EnvVarMgr.EnvVarsByC[apicommon.CoreAgentContainerName]
		assert.True(t, apiutils.IsEqualStruct(nodeCoreAgentEnvVars, wantEnvVars), "Core agent envvars \ndiff = %s", cmp.Diff(nodeCoreAgentEnvVars, wantEnvVars))
	}

	sbomWithContainerImageOverlayFSWantFunc := func(t testing.TB, mgrInterface feature.PodTemplateManagers) {
		mgr := mgrInterface.(*fake.PodTemplateManagers)

		wantEnvVars := []*corev1.EnvVar{
			{
				Name:  DDSBOMEnabled,
				Value: "true",
			},
			{
				Name:  DDSBOMContainerImageEnabled,
				Value: "true",
			},
			{

				Name:  DDSBOMContainerOverlayFSDirectScan,
				Value: "true",
			},
			{
				Name:  DDSBOMHostEnabled,
				Value: "false",
			},
		}

		nodeCoreAgentEnvVars := mgr.EnvVarMgr.EnvVarsByC[apicommon.CoreAgentContainerName]
		assert.True(t, apiutils.IsEqualStruct(nodeCoreAgentEnvVars, wantEnvVars), "Core agent envvars \ndiff = %s", cmp.Diff(nodeCoreAgentEnvVars, wantEnvVars))
	}

	sbomWithHostWantFunc := func(t testing.TB, mgrInterface feature.PodTemplateManagers) {
		mgr := mgrInterface.(*fake.PodTemplateManagers)

		wantCoreAgentsEnvVars := []*corev1.EnvVar{
			{
				Name:  DDSBOMEnabled,
				Value: "true",
			},
			{
				Name:  DDSBOMContainerImageEnabled,
				Value: "false",
			},
			{
				Name:  DDSBOMHostEnabled,
				Value: "true",
			},
			{
				Name:  common.DDHostRootEnvVar,
				Value: common.HostRootMountPath,
			},
		}

		nodeCoreAgentEnvVars := mgr.EnvVarMgr.EnvVarsByC[apicommon.CoreAgentContainerName]
		assert.True(t, apiutils.IsEqualStruct(nodeCoreAgentEnvVars, wantCoreAgentsEnvVars), "Core agent envvars \ndiff = %s", cmp.Diff(nodeCoreAgentEnvVars, wantCoreAgentsEnvVars))

		wantVolumeMounts := []corev1.VolumeMount{
			{
				Name:      common.HostRootVolumeName,
				MountPath: common.HostRootMountPath,
				ReadOnly:  true,
			},
		}

		agentVolumeMounts := mgr.VolumeMountMgr.VolumeMountsByC[apicommon.CoreAgentContainerName]
		assert.True(t, apiutils.IsEqualStruct(agentVolumeMounts, wantVolumeMounts), "Agent volume mounts \ndiff = %s", cmp.Diff(agentVolumeMounts, wantVolumeMounts))

		wantVolumes := []corev1.Volume{
			{
				Name: common.HostRootVolumeName,
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{
						Path: common.HostRootHostPath,
					},
				},
			},
		}

		volumes := mgr.VolumeMgr.Volumes
		assert.True(t, apiutils.IsEqualStruct(volumes, wantVolumes), "Volumes \ndiff = %s", cmp.Diff(volumes, wantVolumes))
	}

	tests := test.FeatureTestSuite{
		{
			Name:          "SBOM not enabled",
			DDA:           sbomDisabled.DeepCopy(),
			WantConfigure: false,
		},
		{
			Name:          "SBOM enabled",
			DDA:           sbomEnabled,
			WantConfigure: true,
			Agent:         test.NewDefaultComponentTest().WithWantFunc(sbomNodeAgentWantFunc),
		},
		{
			Name:          "SBOM enabled, ContainerImage enabled",
			DDA:           sbomEnabledContainerImageEnabled,
			WantConfigure: true,
			Agent:         test.NewDefaultComponentTest().WithWantFunc(sbomWithContainerImageWantFunc),
		},
		{
			Name:          "SBOM enabled, ContainerImage enabled, overlayFS direct scan",
			DDA:           sbomEnabledContainerImageOverlayFSEnabled,
			WantConfigure: true,
			Agent:         test.NewDefaultComponentTest().WithWantFunc(sbomWithContainerImageOverlayFSWantFunc),
		},
		{
			Name:          "SBOM enabled, Host enabled",
			DDA:           sbomEnabledHostEnabled,
			WantConfigure: true,
			Agent:         test.NewDefaultComponentTest().WithWantFunc(sbomWithHostWantFunc),
		},
	}

	tests.Run(t, buildSBOMFeature)
}

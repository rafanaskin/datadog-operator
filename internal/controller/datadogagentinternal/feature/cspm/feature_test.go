// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package cspm

import (
	"fmt"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apicommon "github.com/DataDog/datadog-operator/api/datadoghq/common"
	"github.com/DataDog/datadog-operator/api/datadoghq/v1alpha1"
	"github.com/DataDog/datadog-operator/api/datadoghq/v2alpha1"
	apiutils "github.com/DataDog/datadog-operator/api/utils"
	"github.com/DataDog/datadog-operator/internal/controller/datadogagent/common"
	"github.com/DataDog/datadog-operator/internal/controller/datadogagentinternal/feature"
	"github.com/DataDog/datadog-operator/internal/controller/datadogagentinternal/feature/fake"
	"github.com/DataDog/datadog-operator/internal/controller/datadogagentinternal/feature/test"
	"github.com/DataDog/datadog-operator/pkg/constants"
	"github.com/DataDog/datadog-operator/pkg/controller/utils/comparison"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
)

var customConfig = &v2alpha1.CustomConfig{
	ConfigMap: &v2alpha1.ConfigMapConfig{
		Name: "custom_test",
		Items: []corev1.KeyToPath{
			{
				Key:  "key1",
				Path: "some/path",
			},
		},
	},
}

func Test_cspmFeature_Configure(t *testing.T) {
	ddaCSPMDisabled := v1alpha1.DatadogAgentInternal{
		Spec: v2alpha1.DatadogAgentSpec{
			Features: &v2alpha1.DatadogFeatures{
				CSPM: &v2alpha1.CSPMFeatureConfig{
					Enabled: apiutils.NewBoolPointer(false),
				},
			},
		},
	}
	ddaCSPMEnabled := ddaCSPMDisabled.DeepCopy()
	{
		ddaCSPMEnabled.Spec.Features.CSPM.Enabled = apiutils.NewBoolPointer(true)
		ddaCSPMEnabled.Spec.Features.CSPM.CustomBenchmarks = &v2alpha1.CustomConfig{
			ConfigMap: &v2alpha1.ConfigMapConfig{
				Name: "custom_test",
				Items: []corev1.KeyToPath{
					{
						Key:  "key1",
						Path: "some/path",
					},
				},
			},
		}
		ddaCSPMEnabled.Spec.Features.CSPM.CheckInterval = &metav1.Duration{Duration: 20 * time.Minute}
		ddaCSPMEnabled.Spec.Features.CSPM.HostBenchmarks = &v2alpha1.CSPMHostBenchmarksConfig{Enabled: apiutils.NewBoolPointer(true)}
	}

	tests := test.FeatureTestSuite{

		{
			Name:          "CSPM not enabled",
			DDAI:          ddaCSPMDisabled.DeepCopy(),
			WantConfigure: false,
		},
		{
			Name:          "CSPM enabled",
			DDAI:          ddaCSPMEnabled,
			WantConfigure: true,
			ClusterAgent:  cspmClusterAgentWantFunc(),
			Agent:         cspmAgentNodeWantFunc(),
		},
	}

	tests.Run(t, buildCSPMFeature)
}

func cspmClusterAgentWantFunc() *test.ComponentTest {
	return test.NewDefaultComponentTest().WithWantFunc(
		func(t testing.TB, mgrInterface feature.PodTemplateManagers) {
			mgr := mgrInterface.(*fake.PodTemplateManagers)
			dcaEnvVars := mgr.EnvVarMgr.EnvVarsByC[apicommon.ClusterAgentContainerName]

			want := []*corev1.EnvVar{
				{
					Name:  DDComplianceConfigEnabled,
					Value: "true",
				},
				{
					Name:  DDComplianceConfigCheckInterval,
					Value: "1200000000000",
				},
			}
			assert.True(t, apiutils.IsEqualStruct(dcaEnvVars, want), "DCA envvars \ndiff = %s", cmp.Diff(dcaEnvVars, want))

			wantVolumeMounts := []corev1.VolumeMount{
				{
					Name:      cspmConfigVolumeName,
					MountPath: "/etc/datadog-agent/compliance.d/some/path",
					SubPath:   "some/path",
					ReadOnly:  true,
				},
			}

			volumeMounts := mgr.VolumeMountMgr.VolumeMountsByC[apicommon.ClusterAgentContainerName]
			assert.True(t, apiutils.IsEqualStruct(volumeMounts, wantVolumeMounts), "Cluster Agent volume mounts \ndiff = %s", cmp.Diff(volumeMounts, wantVolumeMounts))

			wantVolumes := []corev1.Volume{
				{
					Name: cspmConfigVolumeName,
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "custom_test",
							},
							Items: []corev1.KeyToPath{{Key: "key1", Path: "some/path"}},
						},
					},
				},
			}
			volumes := mgr.VolumeMgr.Volumes
			assert.True(t, apiutils.IsEqualStruct(volumes, wantVolumes), "Cluster Agent volumes \ndiff = %s", cmp.Diff(volumes, wantVolumes))

			// check annotations
			hash, err := comparison.GenerateMD5ForSpec(customConfig)
			assert.NoError(t, err)
			wantAnnotations := map[string]string{
				fmt.Sprintf(constants.MD5ChecksumAnnotationKey, feature.CSPMIDType): hash,
			}
			annotations := mgr.AnnotationMgr.Annotations
			assert.True(t, apiutils.IsEqualStruct(annotations, wantAnnotations), "Annotations \ndiff = %s", cmp.Diff(annotations, wantAnnotations))

		},
	)
}

func cspmAgentNodeWantFunc() *test.ComponentTest {
	return test.NewDefaultComponentTest().WithWantFunc(
		func(t testing.TB, mgrInterface feature.PodTemplateManagers) {
			mgr := mgrInterface.(*fake.PodTemplateManagers)

			want := []*corev1.EnvVar{
				{
					Name:  DDComplianceConfigEnabled,
					Value: "true",
				},
				{
					Name:  common.DDHostRootEnvVar,
					Value: common.HostRootMountPath,
				},
				{
					Name:  DDComplianceConfigCheckInterval,
					Value: "1200000000000",
				},
				{
					Name:  DDComplianceHostBenchmarksEnabled,
					Value: "true",
				},
			}

			securityAgentEnvVars := mgr.EnvVarMgr.EnvVarsByC[apicommon.SecurityAgentContainerName]
			assert.True(t, apiutils.IsEqualStruct(securityAgentEnvVars, want), "Agent envvars \ndiff = %s", cmp.Diff(securityAgentEnvVars, want))

			// check volume mounts
			wantVolumeMounts := []corev1.VolumeMount{
				{
					Name:      securityAgentComplianceConfigDirVolumeName,
					MountPath: "/etc/datadog-agent/compliance.d",
					ReadOnly:  true,
				},
				{
					Name:      common.CgroupsVolumeName,
					MountPath: common.CgroupsMountPath,
					ReadOnly:  true,
				},
				{
					Name:      common.PasswdVolumeName,
					MountPath: common.PasswdMountPath,
					ReadOnly:  true,
				},
				{
					Name:      common.ProcdirVolumeName,
					MountPath: common.ProcdirMountPath,
					ReadOnly:  true,
				},
				{
					Name:      common.HostRootVolumeName,
					MountPath: common.HostRootMountPath,
					ReadOnly:  true,
				},
				{
					Name:      common.GroupVolumeName,
					MountPath: common.GroupMountPath,
					ReadOnly:  true,
				},
			}

			securityAgentVolumeMounts := mgr.VolumeMountMgr.VolumeMountsByC[apicommon.SecurityAgentContainerName]
			assert.True(t, apiutils.IsEqualStruct(securityAgentVolumeMounts, wantVolumeMounts), "Security Agent volume mounts \ndiff = %s", cmp.Diff(securityAgentVolumeMounts, wantVolumeMounts))

			// check volumes
			wantVolumes := []corev1.Volume{
				{
					Name: cspmConfigVolumeName,
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "custom_test",
							},
							Items: []corev1.KeyToPath{{Key: "key1", Path: "some/path"}},
						},
					},
				},
				{
					Name: securityAgentComplianceConfigDirVolumeName,
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
				{
					Name: common.CgroupsVolumeName,
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: common.CgroupsHostPath,
						},
					},
				},
				{
					Name: common.PasswdVolumeName,
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: common.PasswdHostPath,
						},
					},
				},
				{
					Name: common.ProcdirVolumeName,
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: common.ProcdirHostPath,
						},
					},
				},
				{
					Name: common.HostRootVolumeName,
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: common.HostRootHostPath,
						},
					},
				},
				{
					Name: common.GroupVolumeName,
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: common.GroupHostPath,
						},
					},
				},
			}

			volumes := mgr.VolumeMgr.Volumes
			assert.True(t, apiutils.IsEqualStruct(volumes, wantVolumes), "Volumes \ndiff = %s", cmp.Diff(volumes, wantVolumes))

			// check annotations
			hash, err := comparison.GenerateMD5ForSpec(customConfig)
			assert.NoError(t, err)
			wantAnnotations := map[string]string{
				fmt.Sprintf(constants.MD5ChecksumAnnotationKey, feature.CSPMIDType): hash,
			}
			annotations := mgr.AnnotationMgr.Annotations
			assert.True(t, apiutils.IsEqualStruct(annotations, wantAnnotations), "Annotations \ndiff = %s", cmp.Diff(annotations, wantAnnotations))
		},
	)
}

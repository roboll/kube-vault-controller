// +build !ignore_autogenerated

/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// This file was autogenerated by deepcopy-gen. Do not edit it manually!

package v1alpha1

import (
	v1 "k8s.io/client-go/pkg/api/v1"
	conversion "k8s.io/client-go/pkg/conversion"
	runtime "k8s.io/client-go/pkg/runtime"
	reflect "reflect"
)

func init() {
	SchemeBuilder.Register(RegisterDeepCopies)
}

// RegisterDeepCopies adds deep-copy functions to the given scheme. Public
// to allow building arbitrary schemes.
func RegisterDeepCopies(scheme *runtime.Scheme) error {
	return scheme.AddGeneratedDeepCopyFuncs(
		conversion.GeneratedDeepCopyFunc{Fn: DeepCopy_v1alpha1_ImageReview, InType: reflect.TypeOf(&ImageReview{})},
		conversion.GeneratedDeepCopyFunc{Fn: DeepCopy_v1alpha1_ImageReviewContainerSpec, InType: reflect.TypeOf(&ImageReviewContainerSpec{})},
		conversion.GeneratedDeepCopyFunc{Fn: DeepCopy_v1alpha1_ImageReviewSpec, InType: reflect.TypeOf(&ImageReviewSpec{})},
		conversion.GeneratedDeepCopyFunc{Fn: DeepCopy_v1alpha1_ImageReviewStatus, InType: reflect.TypeOf(&ImageReviewStatus{})},
	)
}

func DeepCopy_v1alpha1_ImageReview(in interface{}, out interface{}, c *conversion.Cloner) error {
	{
		in := in.(*ImageReview)
		out := out.(*ImageReview)
		out.TypeMeta = in.TypeMeta
		if err := v1.DeepCopy_v1_ObjectMeta(&in.ObjectMeta, &out.ObjectMeta, c); err != nil {
			return err
		}
		if err := DeepCopy_v1alpha1_ImageReviewSpec(&in.Spec, &out.Spec, c); err != nil {
			return err
		}
		out.Status = in.Status
		return nil
	}
}

func DeepCopy_v1alpha1_ImageReviewContainerSpec(in interface{}, out interface{}, c *conversion.Cloner) error {
	{
		in := in.(*ImageReviewContainerSpec)
		out := out.(*ImageReviewContainerSpec)
		out.Image = in.Image
		return nil
	}
}

func DeepCopy_v1alpha1_ImageReviewSpec(in interface{}, out interface{}, c *conversion.Cloner) error {
	{
		in := in.(*ImageReviewSpec)
		out := out.(*ImageReviewSpec)
		if in.Containers != nil {
			in, out := &in.Containers, &out.Containers
			*out = make([]ImageReviewContainerSpec, len(*in))
			for i := range *in {
				(*out)[i] = (*in)[i]
			}
		} else {
			out.Containers = nil
		}
		if in.Annotations != nil {
			in, out := &in.Annotations, &out.Annotations
			*out = make(map[string]string)
			for key, val := range *in {
				(*out)[key] = val
			}
		} else {
			out.Annotations = nil
		}
		out.Namespace = in.Namespace
		return nil
	}
}

func DeepCopy_v1alpha1_ImageReviewStatus(in interface{}, out interface{}, c *conversion.Cloner) error {
	{
		in := in.(*ImageReviewStatus)
		out := out.(*ImageReviewStatus)
		out.Allowed = in.Allowed
		out.Reason = in.Reason
		return nil
	}
}

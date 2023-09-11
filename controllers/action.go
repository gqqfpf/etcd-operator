package controllers

import (
	"context"
	"fmt"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Action interface {
	Execute(ctx context.Context) error
}

type PatchStatus struct {
	client   client.Client
	original client.Object
	new      client.Object
}

func (p *PatchStatus) Execute(ctx context.Context) error {
	if reflect.DeepEqual(p.original, p.new) {
		return nil
	}

	// 更新状态
	if err := p.client.Status().Patch(ctx, p.new, client.MergeFrom(p.original)); err != nil {
		return fmt.Errorf("while pathing status error %q\n", err)
	}
	return nil
}

// CreateObject 创建一个 新的资源对象
type CreateObject struct {
	client client.Client
	obj    client.Object
}

func (o *CreateObject) Execute(ctx context.Context) error {
	if err := o.client.Create(ctx, o.obj); err != nil {
		return fmt.Errorf("err %q while createing object", err)
	}
	return nil
}

package controllers

import (
	. "github.com/onsi/gomega"

	"context"
	"fmt"
	"math/rand"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/adambkaplan/plex-operator/api/v1alpha1"
)

func RandomName(baseName string) string {
	return fmt.Sprintf("%s-%s", baseName, strconv.Itoa(rand.Intn(10000)))
}

func InitTestEnvironment(k8sClient client.Client, plex *v1alpha1.PlexMediaServer) (context.Context, *corev1.Namespace) {
	ctx := context.Background()
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: plex.Namespace,
		},
	}
	err := k8sClient.Create(ctx, ns, &client.CreateOptions{})
	Expect(err).NotTo(HaveOccurred())
	err = k8sClient.Create(ctx, plex, &client.CreateOptions{})
	Expect(err).NotTo(HaveOccurred())
	return ctx, ns
}

func TearDownTestEnvironment(ctx context.Context, k8sClient client.Client, plex *v1alpha1.PlexMediaServer, ns *corev1.Namespace) {
	err := k8sClient.Delete(ctx, plex, &client.DeleteOptions{})
	Expect(err).NotTo(HaveOccurred())
	err = k8sClient.Delete(ctx, ns, &client.DeleteOptions{})
	Expect(err).NotTo(HaveOccurred())
}

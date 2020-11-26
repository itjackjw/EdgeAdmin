package acme

import (
	"github.com/TeaOSLab/EdgeAdmin/internal/web/actions/actionutils"
	"github.com/TeaOSLab/EdgeAdmin/internal/web/actions/default/dns/domains/domainutils"
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	"github.com/iwind/TeaGo/actions"
	"github.com/iwind/TeaGo/maps"
	"strings"
)

type CreateAction struct {
	actionutils.ParentAction
}

func (this *CreateAction) Init() {
	this.Nav("", "", "create")
}

func (this *CreateAction) RunGet(params struct{}) {
	// 获取所有可用的用户
	usersResp, err := this.RPC().ACMEUserRPC().FindAllACMEUsers(this.AdminContext(), &pb.FindAllACMEUsersRequest{
		AdminId: this.AdminId(),
		UserId:  0,
	})
	if err != nil {
		this.ErrorPage(err)
		return
	}
	userMaps := []maps.Map{}
	for _, user := range usersResp.AcmeUsers {
		description := user.Description
		if len(description) > 0 {
			description = "（" + description + "）"
		}

		userMaps = append(userMaps, maps.Map{
			"id":          user.Id,
			"description": description,
			"email":       user.Email,
		})
	}
	this.Data["users"] = userMaps

	// 域名解析服务商
	providersResp, err := this.RPC().DNSProviderRPC().FindAllEnabledDNSProviders(this.AdminContext(), &pb.FindAllEnabledDNSProvidersRequest{
		AdminId: this.AdminId(),
		UserId:  0,
	})
	if err != nil {
		this.ErrorPage(err)
		return
	}
	providerMaps := []maps.Map{}
	for _, provider := range providersResp.DnsProviders {
		providerMaps = append(providerMaps, maps.Map{
			"id":       provider.Id,
			"name":     provider.Name,
			"typeName": provider.TypeName,
		})
	}
	this.Data["providers"] = providerMaps

	this.Show()
}

func (this *CreateAction) RunPost(params struct {
	TaskId        int64
	AcmeUserId    int64
	DnsProviderId int64
	DnsDomain     string
	Domains       []string
	AutoRenew     bool

	Must *actions.Must
}) {
	if params.AcmeUserId <= 0 {
		this.Fail("请选择一个申请证书的用户")
	}
	if params.DnsProviderId <= 0 {
		this.Fail("请选择DNS服务商")
	}
	if len(params.DnsDomain) == 0 {
		this.Fail("请输入顶级域名")
	}
	dnsDomain := strings.ToLower(params.DnsDomain)
	if !domainutils.ValidateDomainFormat(dnsDomain) {
		this.Fail("请输入正确的顶级域名")
	}

	if len(params.Domains) == 0 {
		this.Fail("请输入证书域名列表")
	}
	realDomains := []string{}
	for _, domain := range params.Domains {
		domain = strings.ToLower(domain)
		if !strings.HasSuffix(domain, "."+dnsDomain) && domain != dnsDomain {
			this.Fail("证书域名中的" + domain + "和顶级域名不一致")
		}
		realDomains = append(realDomains, domain)
	}

	if params.TaskId == 0 {
		createResp, err := this.RPC().ACMETaskRPC().CreateACMETask(this.AdminContext(), &pb.CreateACMETaskRequest{
			AcmeUserId:    params.AcmeUserId,
			DnsProviderId: params.DnsProviderId,
			DnsDomain:     dnsDomain,
			Domains:       realDomains,
			AutoRenew:     params.AutoRenew,
		})
		if err != nil {
			this.ErrorPage(err)
			return
		}
		params.TaskId = createResp.AcmeTaskId
		defer this.CreateLogInfo("创建证书申请任务 %d", createResp.AcmeTaskId)
	} else {
		_, err := this.RPC().ACMETaskRPC().UpdateACMETask(this.AdminContext(), &pb.UpdateACMETaskRequest{
			AcmeTaskId:    params.TaskId,
			AcmeUserId:    params.AcmeUserId,
			DnsProviderId: params.DnsProviderId,
			DnsDomain:     dnsDomain,
			Domains:       realDomains,
			AutoRenew:     params.AutoRenew,
		})
		if err != nil {
			this.ErrorPage(err)
			return
		}

		defer this.CreateLogInfo("修改证书申请任务 %d", params.TaskId)
	}

	this.Data["taskId"] = params.TaskId

	this.Success()
}
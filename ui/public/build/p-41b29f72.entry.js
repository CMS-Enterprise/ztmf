import{r as t,c as s,h as i,H as h,g as n}from"./p-80f0c69a.js";const o="DISMISSED_BANNERS",a=class{constructor(i){t(this,i),this.componentLibraryAnalytics=s(this,"component-library-analytics",7),this.prepareBannerID=()=>`${this.headline}:${this.el.innerHTML}`,this.dismiss=()=>{const t=this.prepareBannerID();if(!this.dismissedBanners.includes(t)&&this.showClose){const s=[...this.dismissedBanners,t];(this.windowSession?window.sessionStorage:window.localStorage).setItem(o,JSON.stringify(s)),this.dismissedBanners=s,this.disableAnalytics||this.componentLibraryAnalytics.emit({componentName:"va-banner",action:"close",details:{headline:this.headline}})}},this.disableAnalytics=!1,this.showClose=!1,this.headline=void 0,this.type="info",this.visible=!0,this.windowSession=!1,this.dismissedBanners=[]}componentWillLoad(){if(this.showClose){const t=(this.windowSession?window.sessionStorage:window.localStorage).getItem(o);this.dismissedBanners=t?JSON.parse(t):[]}}render(){var t;const s=this.showClose&&(null===(t=this.dismissedBanners)||void 0===t?void 0:t.includes(this.prepareBannerID()));return!this.visible||s?null:i(h,null,i("va-alert",{visible:!0,"full-width":!0,closeable:this.showClose,onCloseEvent:this.showClose?this.dismiss:void 0,status:this.type,"data-role":"banner"},i("h3",{slot:"headline"},this.headline),i("slot",null)))}get el(){return n(this)}};a.style="";export{a as va_banner}
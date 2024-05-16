var r,n,e=(r=function(r){
/*!
	Copyright (c) 2018 Jed Watson.
	Licensed under the MIT License (MIT), see
	http://jedwatson.github.io/classnames
*/
!function(){var n={}.hasOwnProperty;function e(){for(var r=[],i=0;i<arguments.length;i++){var o=arguments[i];if(o){var t=typeof o;if("string"===t||"number"===t)r.push(o);else if(Array.isArray(o)){if(o.length){var u=e.apply(null,o);u&&r.push(u)}}else if("object"===t){if(o.toString!==Object.prototype.toString&&!o.toString.toString().includes("[native code]")){r.push(o.toString());continue}for(var f in o)n.call(o,f)&&o[f]&&r.push(f)}}}return r.join(" ")}r.exports?(e.default=e,r.exports=e):window.classNames=e}()},r(n={path:undefined,exports:{},require:function(){return function(){throw new Error("Dynamic requires are not currently supported by @rollup/plugin-commonjs")}()}}),n.exports);export{e as c}